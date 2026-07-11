# Performance tool

measure min, max, mean ± σ, median, p95

## WHAT TO MEASURE:

- Actual time from start to finish
- CPU time: user, system, all
- RAM: virtual, resident
- num of processes and threads
- open file descriptors
- size of binary / script on disk
- size with all child processses / daemons binaries / dynamic libs etc on disk

## WHAT ELSE TO RECORD:

host system: 

- cpu, cpu usage before measure, num of running processes
- RAM, RAM usage before measure, after measure
- disk ?
- datatime of measurement
- name of tool
- sha of tool, version if available
- params of measurement

## HOW TO MEASURE:

- single tool mode
- compare tools mode (any amount)
- compare runs mode
- setup prewarm
- setup how many times to run
- setup what to mesure

## WHAT TO REPORT:

- report to stdout
- offer options to render svg, html
- charts
- to save as just txt
- to save as markdown with svg charts
- single tool report
- compare multiple tools report
- compare tool in time (across versions etc) report
