.\ab -k -c 1 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0001_parallel.txt
.\ab -k -c 2 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0002_parallel.txt
.\ab -k -c 4 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0004_parallel.txt
.\ab -k -c 8 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0008_parallel.txt
.\ab -k -c 16 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0016_parallel.txt
.\ab -k -c 32 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0032_parallel.txt
.\ab -k -c 64 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0064_parallel.txt
.\ab -k -c 128 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0128_parallel.txt
.\ab -k -c 256 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0256_parallel.txt
.\ab -k -c 512 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_0512_parallel.txt
.\ab -k -c 1024 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_1024_parallel.txt
.\ab -k -c 2048 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_2048_parallel.txt
.\ab -k -c 4096 -n 100000 "http://127.0.0.1:3000/depts/random" > ab_4096_parallel.txt
rem .\ab -k -c 2 -n 1000000 "http://127.0.0.1:3000/depts/10"