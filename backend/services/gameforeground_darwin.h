// Bringing the game to the front on macOS (gameforeground_darwin.go): what Go
// calls.
//
// Plain C types only, because cgo copies this file into the Go preamble.
#ifndef KAPITAL_SERVICES_GAMEFOREGROUND_DARWIN_H
#define KAPITAL_SERVICES_GAMEFOREGROUND_DARWIN_H

// kapitalActivateProcess activates the running application with this pid, so
// its frontmost window takes the keyboard. It returns 1 when macOS accepted the
// request and 0 when there is no such application or it refused.
int kapitalActivateProcess(int pid);

#endif
