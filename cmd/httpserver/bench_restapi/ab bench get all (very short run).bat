.\ab -k -c 1 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0001_parallel_very_shot.txt
.\ab -k -c 2 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0002_parallel_very_shot.txt
.\ab -k -c 4 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0004_parallel_very_shot.txt
.\ab -k -c 8 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0008_parallel_very_shot.txt
.\ab -k -c 16 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0016_parallel_very_shot.txt
.\ab -k -c 32 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0032_parallel_very_shot.txt
.\ab -k -c 64 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0064_parallel_very_shot.txt
.\ab -k -c 128 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0128_parallel_very_shot.txt
.\ab -k -c 256 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0256_parallel_very_shot.txt
.\ab -k -c 512 -n 1000 "http://127.0.0.1:3000/depts/random" > ab_0512_parallel_very_shot.txt
rem .\ab -k -c 2 -n 1000000 "http://127.0.0.1:3000/depts/10"