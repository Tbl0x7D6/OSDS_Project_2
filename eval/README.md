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
python3 eval/perf.py --counts 1,3,5,7 --difficulties 3,4 --duration 10
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
