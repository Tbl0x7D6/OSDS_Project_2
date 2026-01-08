# Performance Evaluation

This folder contains an end-to-end performance evaluator for the Go blockchain miners.

The evaluator:
- deploys miners via `make deploy_miner` with different `COUNT`/`DIFFICULTY`
- measures chain growth over a fixed window by polling miners using the Go client (`./bin/client blockchain` → `chain_length`)
- writes one output folder per run and generates a plot automatically

## Prerequisites

- `minerip.txt` exists in the repo root and contains miner IPs (one per line). The evaluator uses the **prefix** of this list (first `COUNT` lines).
- You can SSH to those miners as `root@<ip>` (same assumption as `make deploy_miner`).
- Port `8001` is reachable from this machine to the miners (RPC).

## Run

From the repo root:

```bash
python3 eval/perf.py --counts 1,3,5,7 --difficulties 10,14,17,19,22 --duration 60 --progress-interval 5 --stop-between
```

Options:
- `--counts` comma-separated miner counts
- `--difficulties` comma-separated difficulties
- `--duration` measurement window in seconds
- `--stop-between` stops miners between each experiment (slower but cleaner)
- `--out-dir` base output directory (default: `logs/perf`)

## Outputs

Each run writes into a timestamped folder:

```
logs/perf/<YYYYMMDD_HHMMSS>/
  results.json
  results.csv
  plot.png
```

Plot details:
- grouped **bar** chart
- x-axis: difficulty
- y-axis: blocks mined (log scale)
- bars grouped by `miner_count`, value label above each bar

## Deploy logs

`make deploy_miner` stores deploy logs under:

```
logs/deploy/deploy_<timestamp>.log
```


## New feature

Done

Added intermediate “watch” output during the measurement window in perf.py: it calls ./bin/client blockchain -miner <addr> every N seconds and overwrites a single console line (no scrolling).
New flag: --progress-interval (default 10.0; set 0 to disable).
What you’ll see

While the test is running, it prints one line like: t=  10s chain_length=123 (+45) and keeps updating it in-place, then prints a newline at the end of the window.
Tested

Ran: python3 [perf.py](http://_vscodecontentref_/9) --counts 1 --difficulties 3 --duration 12 --progress-interval 2 --stop-between
Confirmed the intermediate line updates and the run completes, writing outputs under logs/perf/<timestamp>/.
If you want the progress line to include difficulty/miner count (e.g., prefix [COUNT=.. DIFF=..]), say so and I’ll add it.