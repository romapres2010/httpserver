.\ab -T application/json -u null.txt -k -c 1 -n 10000 "http://127.0.0.1:3000/depts/random" > ab_0001_parallel_randomdeptupdate.txt
.\ab -T application/json -u null.txt -k -c 2 -n 10000 "http://127.0.0.1:3000/depts/random" > ab_0002_parallel_randomdeptupdate.txt
.\ab -T application/json -u null.txt -k -c 4 -n 10000 "http://127.0.0.1:3000/depts/random" > ab_0004_parallel_randomdeptupdate.txt
.\ab -T application/json -u null.txt -k -c 8 -n 10000 "http://127.0.0.1:3000/depts/random" > ab_0008_parallel_randomdeptupdate.txt
rem .\ab -k -c 2 -n 1000000 "http://127.0.0.1:3000/depts/10"