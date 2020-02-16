rem set GOGCTRACE=1
rem set GODEBUG=gctrace=1
.\httpserver.exe -httpcfg test_config.cfg -l 127.0.0.1:3000 -httpu roman -httppwd Welcome1 -jwtk qwerlkc8SFlwe -d INFO -log test.log 