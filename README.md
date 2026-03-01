## Description
A Go-based application that captures user-selected text anywhere on their computer and streams back the corresponding speech in real time.

##   Run project :

Add .env to each folder

#### FRONT
`go get github.com/getlantern/systray`
`go mod tidy`

#### BACK
`go mod tidy`


## TODO
- [x] create separation with front and back
- [x] test with real api (google)
- [x] add guards to limit text size
- [ ] add authentication
- [ ] add api key to request -> manage abort request per client
- [ ] add progress bar -> audio file player
- [ ] add floating icon ?
