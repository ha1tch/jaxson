# Shaxon against pyshacl and Apache Jena

Host: python 3.13.16, platform Linux-6.18.44-fc-v70-x86_64-with-glibc2.39, machine x86_64, cpus 2, cpu Intel(R) Xeon(R) Processor @ 2.80GHz, go go1.27.1 linux/amd64, java openjdk version "21.0.12.1" 2026-08-18, pyshacl 0.40.1, rdflib 7.6.0  

Times are milliseconds. End to end is the JSON input to a verdict (Shaxon: the whole process; pyshacl and Jena include the RDF lift, and Jena the JVM start). Engine only is validation alone on data already in each engine's form, best of several runs after a warm-up. "skipped" means the harness's time budget projected that cell too slow to run.

## rolling-quota

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 473 | 5.5 | 85 | 1,185 | 15x faster | 214x faster |
| 1,000 | 4073 | 4.5 | 205 | 1,373 | 45x faster | 303x faster |
| 4,000 | 16073 | 7.0 | 708 | 2,492 | 101x faster | 356x faster |
| 20,000 | 80073 | 31 | 4,556 | 6,708 | 147x faster | 216x faster |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 473 | 0.31 | 77 | 8.5 | 245x faster | 27x faster | 12 |
| 1,000 | 4073 | 0.28 | 117 | 16 | 426x faster | 59x faster | 183 |
| 4,000 | 16073 | 0.84 | 350 | 12 | 416x faster | 14x faster | 383 |
| 20,000 | 80073 | 3.0 | 2,422 | 25 | 814x faster | 8.3x faster | 2,451 |

## chinese-wall

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 411 | 3.6 | 37 | 1,246 | 10x faster | 346x faster |
| 1,000 | 4011 | 7.7 | 223 | 1,406 | 29x faster | 182x faster |
| 4,000 | 16011 | 16 | 535 | 1,812 | 33x faster | 113x faster |
| 20,000 | 80011 | 58 | 3,161 | 4,638 | 55x faster | 80x faster |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 411 | 0.20 | 22 | 3.3 | 112x faster | 17x faster | 11 |
| 1,000 | 4011 | 1.5 | 91 | 3.7 | 61x faster | 2.5x faster | 66 |
| 4,000 | 16011 | 9.8 | 300 | 3.7 | 31x faster | 2.7x slower | 243 |
| 20,000 | 80011 | 31 | 1,696 | 9.3 | 54x faster | 3.4x slower | 1,397 |

## four-eyes-release

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 1546 | 5.7 | 80 | 1,084 | 14x faster | 189x faster |
| 1,000 | 15101 | 137 | 2,338 | 1,276 | 17x faster | 9.3x faster |
| 4,000 | 60339 | 2,826 | 37,626 | 2,062 | 13x faster | 1.4x slower |
| 20,000 | 301568 | 80,411 | skipped | 6,475 | - | 12x slower |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 1546 | 1.7 | 59 | 7.1 | 35x faster | 4.2x faster | 12 |
| 1,000 | 15101 | 122 | 2,019 | 6.5 | 17x faster | 19x slower | 119 |
| 4,000 | 60339 | 2,851 | 35,430 | 8.2 | 12x faster | 349x slower | 548 |
| 20,000 | 301568 | 79,334 | skipped | 19 | - | 4079x slower | - |

## delegation-chain

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 281 | 3.1 | 77 | 1,093 | 25x faster | 350x faster |
| 1,000 | 2081 | 5.4 | 363 | 1,363 | 67x faster | 250x faster |
| 4,000 | 8081 | 16 | 1,234 | 2,107 | 78x faster | 133x faster |
| 20,000 | 40081 | 97 | 7,129 | 5,359 | 74x faster | 55x faster |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 281 | 0.32 | 82 | 12 | 257x faster | 36x faster | 8.3 |
| 1,000 | 2081 | 1.9 | 261 | 7.6 | 138x faster | 4.0x faster | 152 |
| 4,000 | 8081 | 4.1 | 1,071 | 16 | 263x faster | 4.0x faster | 354 |
| 20,000 | 40081 | 28 | 5,657 | 12 | 201x faster | 2.3x slower | 1,619 |

## break-glass

### End to end

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena |
|---|---|---|---|---|---|---|
| 100 | 1969 | 6.8 | 1,304 | 1,377 | 192x faster | 203x faster |
| 1,000 | 19398 | 326 | 11,759 | 1,929 | 36x faster | 5.9x faster |
| 4,000 | 77617 | 4,811 | 52,741 | 4,115 | 11x faster | 1.2x slower |
| 20,000 | 387551 | 137,036 | skipped | 9,543 | - | 14x slower |

### Engine only

| Events | Steps | Shaxon | pyshacl | Jena | Shaxon vs pyshacl | Shaxon vs Jena | RDF lift (not counted) |
|---|---|---|---|---|---|---|---|
| 100 | 1969 | 2.1 | 1,105 | 36 | 531x faster | 17x faster | 11 |
| 1,000 | 19398 | 290 | 12,240 | 68 | 42x faster | 4.2x slower | 118 |
| 4,000 | 77617 | 4,583 | 50,676 | 144 | 11x faster | 32x slower | 447 |
| 20,000 | 387551 | 147,696 | skipped | 945 | - | 156x slower | - |

## Agreement

Every cell where more than one engine ran returned the same allow/deny verdict (and, end to end, the same reasons).
