1. go to `constants/constants.go` and edit `User` field for blunos
2. go to `constants/blunolist.go` and comment out inactive blunos
3. open two terminals, L and R
4. L terminal should run `docker-compose up` first
5. R terminal should run `docker exec -it comms-int_laptop_client_1 /bin/bash` afterwards
6. R terminal should run `python mock_client.py` when L client is ready to receive a connection
7. On R terminal, type in `0` and press enter when ready to start golang app
8. When done, `ctrl-c` only on left terminal. WAIT for graceful exit (files get renamed at this point)
