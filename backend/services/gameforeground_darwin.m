// Bringing the game to the front on macOS at the loading card's handover
// (gameforeground_darwin.go, #258). Compiled with ARC, like activity_darwin.m.
//
// The game's window is never touched: the application the game's Java process
// is (GLFW makes it one) is asked to activate, by the pid the tracker found. It
// reads nothing of the game and needs no permission.

#import <AppKit/AppKit.h>

#include "gameforeground_darwin.h"

// This runs on a Go thread, which has no autorelease pool of its own.

int kapitalActivateProcess(int pid) {
    @autoreleasepool {
        if (pid <= 0) return 0;
        // https://developer.apple.com/documentation/appkit/nsrunningapplication/init(processidentifier:)
        NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:(pid_t)pid];
        if (app == nil || app.terminated) return 0;
        // The macOS 14 branch is compiled only by an SDK that declares its call:
        // the macOS 13 SDK, the newest a 2017 iMac's Command Line Tools have,
        // does not, and refuses the whole file. Release builds use a newer SDK
        // and keep the branch.
#if defined(__MAC_OS_X_VERSION_MAX_ALLOWED) && __MAC_OS_X_VERSION_MAX_ALLOWED >= 140000
        if (@available(macOS 14.0, *)) {
            // macOS 14 activates cooperatively: the active app yields first, and
            // the activation takes no options.
            // https://developer.apple.com/documentation/appkit/nsapplication/yieldactivation(to:)
            [NSApp yieldActivationToApplication:app];
            return [app activateWithOptions:0] ? 1 : 0;
        }
#endif
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
        // Deprecated in macOS 14, which takes the branch above when it is compiled.
        return [app activateWithOptions:NSApplicationActivateIgnoringOtherApps] ? 1 : 0;
#pragma clang diagnostic pop
    }
}
