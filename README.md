# Trying to make a mangadex importer

cli
```bash
go run main.go
```
web dev
```bash
cd web
go run -tags dev .
```
web deploy
```bash
cd web
go build .
./web
```

todo
- [ ] handle duplicate titles
- [ ] add user imput
- [ ] add progress bar 
- [ ] add logging
- [x] add webserver
- [x] solve ratelimiter
- [ ] queue
- [ ] add web ui
