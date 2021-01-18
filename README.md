For test/debug you can run sleepserver by

```go run sleepserver/main.go```

it will listen 8081 port and receive requests of type
`http://127.0.0.1:8081/{key}`, where key is duration.
Key will be parsed by time.ParseDuration func and if it fails - server return 500 error
if success - handler will sleep for passed duration

To run target server - go run main.go

Also i indluded requests.http file. You can install http plugin to VScode to be able to fire requests from it. OR just use 
jetbrains IDE, it comes out of the box (i'm using inteleji idea so it works)