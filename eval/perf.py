#!/usr/bin/env python3

"""End-to-end performance evaluation + plotting.

Runs `make deploy_miner COUNT=<n> DIFFICULTY=<d>` over a grid and measures
how many blocks are mined in a fixed time window. After evaluation, it
automatically writes results (JSON/CSV) and generates a grouped bar chart.

Chart:
- x-axis: difficulty
- bars (grouped by x): miner count
- y-axis: blocks mined (log scale)
- annotation: small-font blocks mined over each bar

This script uses `./bin/client blockchain -miner <ip>:8001` and reads `chain_length`.
"""

from __future__ import annotations

import argparse
import csv
import json
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import matplotlib.pyplot as plt


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_COUNTS = [1, 3, 5, 7]
DEFAULT_DIFFICULTIES = [3, 4, 5]
DEFAULT_MINER_PORT = 8001


@dataclass(frozen=True)
class RunResult:
    count: int
    difficulty: int
    duration_sec: int
    ips: List[str]
    start_chain_length: int
    end_chain_length: int
    blocks_mined: int
    deploy_elapsed_sec: float
    started_at: str
    ended_at: str


def run_cmd(
    args: List[str],
    *,
    cwd: Path,
    timeout_sec: float,
) -> subprocess.CompletedProcess:
    return subprocess.run(
        args,
        cwd=str(cwd),
        timeout=max(1, int(timeout_sec)),
        check=True,
        text=True,
        capture_output=True,
    )


def read_miner_ips(path: Path) -> List[str]:
    if not path.exists():
        raise FileNotFoundError(f"Missing {path}. This script expects miner IPs in minerip.txt.")

    ips: List[str] = []
    for line in path.read_text(encoding="utf-8").splitlines():
        ip = line.strip()
        if not ip or ip.startswith("#"):
            continue
        ips.append(ip)
    return ips


def client_chain_length(miner_addr: str, *, timeout_sec: float) -> Optional[int]:
    client_bin = REPO_ROOT / "bin" / "client"
    if not client_bin.exists():
        raise FileNotFoundError(
            f"Missing {client_bin}. Run `make compile` (or let `make deploy_miner` build it) first."
        )

    try:
        cp = run_cmd(
            [str(client_bin), "blockchain", "-miner", miner_addr],
            cwd=REPO_ROOT,
            timeout_sec=timeout_sec,
        )
        data = json.loads(cp.stdout)
        if "chain_length" not in data:
            return None
        return int(data["chain_length"])
    except Exception:
        return None


def wait_for_chain_length(
    miner_ip: str,
    *,
    port: int,
    timeout_sec: float,
    poll_interval_sec: float,
) -> int:
    """Wait until a single miner responds to `client blockchain` and return its chain_length."""
    deadline = time.time() + timeout_sec
    addr = f"{miner_ip}:{port}"
    while time.time() < deadline:
        remaining = deadline - time.time()
        per_call_timeout = max(0.5, min(2.0, remaining))
        cur = client_chain_length(addr, timeout_sec=per_call_timeout)
        if cur is not None:
            return cur
        time.sleep(poll_interval_sec)
    raise TimeoutError(f"Timed out waiting for miner RPC at {addr}")


def pick_observer_ip(ips: List[str]) -> str:
    if not ips:
        raise ValueError("No miner IPs provided")
    # Use the first miner in the deployed prefix as the single observation point.
    return ips[0]


def make_deploy_miner(*, count: int, difficulty: int) -> None:
    subprocess.run(
        ["make", "deploy_miner", f"COUNT={count}", f"DIFFICULTY={difficulty}"],
        cwd=str(REPO_ROOT),
        check=True,
    )


def make_stop_miner() -> None:
    subprocess.run(["make", "stop_miner"], cwd=str(REPO_ROOT), check=False)


def run_experiment(
    *,
    all_ips: List[str],
    count: int,
    difficulty: int,
    duration_sec: int,
    port: int,
    warmup_sec: float,
    ready_timeout_sec: float,
    poll_interval_sec: float,
    progress_interval_sec: float,
) -> RunResult:
    if count > len(all_ips):
        raise ValueError(f"Requested COUNT={count} but only {len(all_ips)} IPs are in minerip.txt")

    ips = all_ips[:count]
    observer_ip = pick_observer_ip(ips)
    started_at = datetime.utcnow().isoformat() + "Z"

    deploy_t0 = time.time()
    make_deploy_miner(count=count, difficulty=difficulty)
    deploy_elapsed = time.time() - deploy_t0

    if warmup_sec > 0:
        time.sleep(warmup_sec)

    # Measure against a single miner via the client.
    # Note: For large COUNT, miners begin mining as soon as they start; because deploy_miner
    # starts miners sequentially, some blocks may already exist by the time deployment finishes.
    start_len = wait_for_chain_length(
        observer_ip,
        port=port,
        timeout_sec=ready_timeout_sec,
        poll_interval_sec=poll_interval_sec,
    )

    # During the measurement window, optionally sample intermediate chain lengths.
    # We overwrite a single console line to avoid noisy output.
    addr = f"{observer_ip}:{port}"
    if progress_interval_sec and progress_interval_sec > 0 and duration_sec > 0:
        last_print_len = 0

        def print_progress_line(msg: str) -> None:
            nonlocal last_print_len
            padded = msg
            if last_print_len > len(msg):
                padded = msg + (" " * (last_print_len - len(msg)))
            last_print_len = len(padded)
            sys.stdout.write("\r" + padded)
            sys.stdout.flush()

        t0 = time.time()
        deadline = t0 + duration_sec
        last_seen = start_len
        print_progress_line(f"  t=   0s chain_length={start_len} (+0)")

        while True:
            now = time.time()
            remaining = deadline - now
            if remaining <= 0:
                break
            time.sleep(min(progress_interval_sec, remaining))
            elapsed = int(time.time() - t0)

            cur = client_chain_length(addr, timeout_sec=2.0)
            if cur is not None:
                last_seen = cur
            print_progress_line(
                f"  t={elapsed:>4}s chain_length={last_seen} (+{last_seen - start_len})"
            )

        sys.stdout.write("\n")
        sys.stdout.flush()
    else:
        time.sleep(duration_sec)

    end_len = wait_for_chain_length(
        observer_ip,
        port=port,
        timeout_sec=ready_timeout_sec,
        poll_interval_sec=poll_interval_sec,
    )

    ended_at = datetime.utcnow().isoformat() + "Z"

    return RunResult(
        count=count,
        difficulty=difficulty,
        duration_sec=duration_sec,
        ips=ips,
        start_chain_length=start_len,
        end_chain_length=end_len,
        blocks_mined=end_len - start_len,
        deploy_elapsed_sec=deploy_elapsed,
        started_at=started_at,
        ended_at=ended_at,
    )


def write_outputs(results: List[RunResult], out_dir: Path, *, ts: str) -> Tuple[Path, Path]:
    out_dir.mkdir(parents=True, exist_ok=True)
    json_path = out_dir / "results.json"
    csv_path = out_dir / "results.csv"

    json_path.write_text(
        json.dumps([r.__dict__ for r in results], indent=2) + "\n",
        encoding="utf-8",
    )

    with csv_path.open("w", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(
            f,
            fieldnames=[
                "count",
                "difficulty",
                "duration_sec",
                "start_chain_length",
                "end_chain_length",
                "blocks_mined",
                "deploy_elapsed_sec",
                "ips",
                "started_at",
                "ended_at",
            ],
        )
        w.writeheader()
        for r in results:
            row = r.__dict__.copy()
            row["ips"] = ",".join(r.ips)
            w.writerow(row)

    return json_path, csv_path


def plot_grouped_bars(
    results: List[RunResult],
    *,
    out_path: Path,
    title: str,
    counts: List[int],
    difficulties: List[int],
) -> None:
    # Build lookup: (difficulty, count) -> blocks
    lookup: Dict[Tuple[int, int], int] = {}
    for r in results:
        lookup[(r.difficulty, r.count)] = r.blocks_mined

    # Grouped bar positions
    x_positions = list(range(len(difficulties)))
    n_groups = len(counts)
    if n_groups == 0:
        raise ValueError("No counts provided")

    total_width = 0.8
    bar_width = total_width / n_groups

    plt.figure(figsize=(9, 5.5))

    for i, count in enumerate(counts):
        offsets = [x - total_width / 2 + (i + 0.5) * bar_width for x in x_positions]
        ys: List[int] = []
        for diff in difficulties:
            ys.append(int(lookup.get((diff, count), 0)))

        bars = plt.bar(offsets, ys, width=bar_width, label=f"miners={count}")

        # Annotate values (small font) above each bar
        for b, y in zip(bars, ys):
            # Place at y, but keep it visible for y=0
            y_text = y if y > 0 else 0.8
            plt.text(
                b.get_x() + b.get_width() / 2,
                y_text,
                str(y),
                ha="center",
                va="bottom",
                fontsize=8,
                rotation=0,
            )

    plt.xlabel("difficulty")
    plt.ylabel("blocks mined (log scale)")
    plt.yscale("log")
    plt.xticks(x_positions, [str(d) for d in difficulties])
    plt.grid(True, which="both", axis="y", linestyle="--", linewidth=0.6, alpha=0.6)
    plt.title(title)
    plt.legend()

    out_path.parent.mkdir(parents=True, exist_ok=True)
    plt.tight_layout()
    plt.savefig(out_path, dpi=150)
    plt.close()


def parse_csv_int_list(s: str) -> List[int]:
    return [int(x.strip()) for x in s.split(",") if x.strip()]


def main() -> int:
    p = argparse.ArgumentParser(description="Run performance eval and plot grouped bar chart")
    p.add_argument("--duration", type=int, default=60)
    p.add_argument("--counts", type=str, default=",".join(str(x) for x in DEFAULT_COUNTS))
    p.add_argument("--difficulties", type=str, default=",".join(str(x) for x in DEFAULT_DIFFICULTIES))
    p.add_argument("--port", type=int, default=DEFAULT_MINER_PORT)
    p.add_argument("--warmup", type=float, default=0.0)
    p.add_argument("--ready-timeout", type=float, default=45.0)
    p.add_argument("--poll-interval", type=float, default=1.0)
    p.add_argument(
        "--progress-interval",
        type=float,
        default=10.0,
        help="Seconds between intermediate client chain_length samples during the window (0 to disable)",
    )
    p.add_argument(
        "--out-dir",
        type=str,
        default=str(REPO_ROOT / "logs" / "perf"),
        help="Where to write JSON/CSV/PNG (default: logs/perf)",
    )
    p.add_argument(
        "--stop-between",
        action="store_true",
        help="Stop miners between each experiment (slower, cleaner)",
    )

    args = p.parse_args()

    counts = parse_csv_int_list(args.counts)
    diffs = parse_csv_int_list(args.difficulties)

    base_out_dir = Path(args.out_dir)
    if not base_out_dir.is_absolute():
        base_out_dir = (REPO_ROOT / base_out_dir).resolve()

    all_ips = read_miner_ips(REPO_ROOT / "minerip.txt")

    ts = datetime.utcnow().strftime("%Y%m%d_%H%M%S")
    run_dir = (base_out_dir / ts).resolve()
    run_dir.mkdir(parents=True, exist_ok=True)

    results: List[RunResult] = []
    try:
        for diff in diffs:
            for count in counts:
                print(f"\n=== Running: COUNT={count}, DIFFICULTY={diff}, duration={args.duration}s ===")
                res = run_experiment(
                    all_ips=all_ips,
                    count=count,
                    difficulty=diff,
                    duration_sec=args.duration,
                    port=args.port,
                    warmup_sec=args.warmup,
                    ready_timeout_sec=args.ready_timeout,
                    poll_interval_sec=args.poll_interval,
                    progress_interval_sec=args.progress_interval,
                )
                results.append(res)
                print(
                    f"Result: blocks_mined={res.blocks_mined} (chain_length {res.start_chain_length} -> {res.end_chain_length}), deploy_elapsed={res.deploy_elapsed_sec:.1f}s"
                )
                if args.stop_between:
                    make_stop_miner()
    finally:
        # Always stop miners at the end.
        make_stop_miner()

    json_path, csv_path = write_outputs(results, run_dir, ts=ts)

    png_path = run_dir / "plot.png"
    title = f"Blocks mined in {args.duration}s (log scale)\n(grouped by miner_count)"
    plot_grouped_bars(
        results,
        out_path=png_path,
        title=title,
        counts=counts,
        difficulties=diffs,
    )

    print("\nOutputs")
    print(f"Dir:  {run_dir}")
    print(f"JSON: {json_path}")
    print(f"CSV:  {csv_path}")
    print(f"PNG:  {png_path}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
