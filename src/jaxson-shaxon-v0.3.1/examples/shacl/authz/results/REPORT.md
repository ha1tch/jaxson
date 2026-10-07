# Shaxon against pyshacl and Apache Jena

Host: python 3.13.16, platform Linux-6.18.44-fc-v77-x86_64-with-glibc2.39, machine x86_64, cpus 2, cpu Intel(R) Xeon(R) Processor @ 2.80GHz, go go1.27.1 linux/amd64, java openjdk version "21.0.12.1" 2026-08-18, pyshacl 0.40.1, rdflib 7.6.0  

Times are milliseconds. End to end is the JSON input to a verdict (Shaxon: the whole process; pyshacl and Jena include the RDF lift, and Jena the JVM start). Engine only is validation alone on data already in each engine's form, best of several runs after a warm-up. "skipped" means the harness's time budget projected that cell too slow to run.

## rolling-quota

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 473 | 4.2 | 73 | 1,366 | 17x faster | 324x faster |
| 1,000 | 4073 | 5.7 | 226 | 1,708 | 40x faster | 302x faster |
| 4,000 | 16073 | 8.5 | 813 | 2,814 | 96x faster | 332x faster |
| 20,000 | 80073 | 49 | 5,446 | 7,427 | 112x faster | 153x faster |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 473 | 0.21 | 66 | 7.5 | 310x faster | 35x faster | 9.2 |
| 1,000 | 4073 | 0.37 | 138 | 16 | 372x faster | 42x faster | 178 |
| 4,000 | 16073 | 0.82 | 447 | 19 | 548x faster | 23x faster | 443 |
| 20,000 | 80073 | 3.6 | 2,523 | 63 | 700x faster | 18x faster | 2,447 |

## chinese-wall

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 411 | 4.3 | 44 | 1,219 | 10x faster | 286x faster |
| 1,000 | 4011 | 5.5 | 148 | 1,524 | 27x faster | 276x faster |
| 4,000 | 16011 | 10 | 523 | 2,025 | 52x faster | 200x faster |
| 20,000 | 80011 | 42 | 3,548 | 5,668 | 84x faster | 135x faster |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 411 | 0.20 | 27 | 3.1 | 133x faster | 16x faster | 14 |
| 1,000 | 4011 | 0.82 | 85 | 4.3 | 104x faster | 5.3x faster | 90 |
| 4,000 | 16011 | 3.3 | 303 | 3.8 | 92x faster | 1.2x faster | 226 |
| 20,000 | 80011 | 10 | 1,934 | 7.6 | 187x faster | 1.4x slower | 1,434 |

## four-eyes-release

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 1546 | 4.7 | 145 | 1,149 | 31x faster | 246x faster |
| 1,000 | 15101 | 6.4 | 2,622 | 1,593 | 411x faster | 250x faster |
| 4,000 | 60339 | 18 | 42,964 | 2,437 | 2336x faster | 132x faster |
| 20,000 | 301568 | 92 | skipped | 7,291 | - | 80x faster |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 1546 | 0.32 | 71 | 5.1 | 223x faster | 16x faster | 8.7 |
| 1,000 | 15101 | 1.4 | 2,312 | 5.6 | 1661x faster | 4.0x faster | 153 |
| 4,000 | 60339 | 6.9 | 38,740 | 12 | 5632x faster | 1.8x faster | 427 |
| 20,000 | 301568 | 31 | skipped | 36 | - | 1.2x faster | - |

## delegation-chain

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 281 | 4.3 | 122 | 1,084 | 28x faster | 251x faster |
| 1,000 | 2081 | 5.1 | 408 | 1,374 | 80x faster | 268x faster |
| 4,000 | 8081 | 13 | 1,461 | 2,057 | 110x faster | 154x faster |
| 20,000 | 40081 | 49 | 8,297 | 6,663 | 171x faster | 137x faster |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 281 | 0.49 | 87 | 12 | 178x faster | 26x faster | 18 |
| 1,000 | 2081 | 0.88 | 326 | 10.0 | 368x faster | 11x faster | 76 |
| 4,000 | 8081 | 3.2 | 1,224 | 8.5 | 386x faster | 2.7x faster | 418 |
| 20,000 | 40081 | 19 | 6,223 | 20 | 319x faster | 1.0x faster | 1,945 |

## break-glass

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 1969 | 7.7 | 1,717 | 1,595 | 222x faster | 206x faster |
| 1,000 | 19398 | 22 | 14,148 | 2,541 | 636x faster | 114x faster |
| 4,000 | 77617 | 80 | 63,156 | 4,375 | 789x faster | 55x faster |
| 20,000 | 387551 | 468 | skipped | 12,180 | - | 26x faster |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 1969 | 1.2 | 1,220 | 26 | 1013x faster | 21x faster | 7.8 |
| 1,000 | 19398 | 14 | 13,200 | 87 | 967x faster | 6.4x faster | 120 |
| 4,000 | 77617 | 67 | 56,231 | 275 | 840x faster | 4.1x faster | 490 |
| 20,000 | 387551 | 334 | skipped | 967 | - | 2.9x faster | - |

## Agreement

Every cell where more than one engine ran returned the same allow/deny verdict (and, end to end, the same reasons).
