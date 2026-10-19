#import <Cocoa/Cocoa.h>
#include "chrome_darwin.h"

// The window Wails made, told by the toolbar main.go asks for
// (TitleBarHiddenInset). The loading card's window has none.
static NSWindow *launcherWindow(void) {
    for (NSWindow *w in [NSApp windows]) {
        if ([[[w toolbar] identifier] isEqualToString:@"wails.toolbar"]) {
            return w;
        }
    }
    return nil;
}

int kapitalWindowChrome(double *titlebar, double *controlsEnd) {
    __block int found = 0;
    __block double height = 0;
    __block double end = 0;
    void (^measure)(void) = ^{
        NSWindow *w = launcherWindow();
        if (w == nil) {
            return;
        }
        // Wails leaves the style automatic, which macOS 15 resolves to the
        // compact bar and macOS 26 to the taller one. Unified is the same
        // family on both: the bar a toolbar window gets in Finder or Safari.
        if (w.toolbarStyle != NSWindowToolbarStyleUnified) {
            w.toolbarStyle = NSWindowToolbarStyleUnified;
            [w.contentView.superview layoutSubtreeIfNeeded];
        }
        NSButton *close = [w standardWindowButton:NSWindowCloseButton];
        NSButton *zoom = [w standardWindowButton:NSWindowZoomButton];
        if (close == nil || zoom == nil) {
            return;
        }
        // Window coordinates run up from the bottom edge, and the full-size
        // content view puts the bar inside the frame, so the bar is twice the
        // lights' centre measured down from the top.
        NSRect c = [close convertRect:close.bounds toView:nil];
        NSRect z = [zoom convertRect:zoom.bounds toView:nil];
        height = 2 * (NSHeight(w.frame) - NSMidY(c));
        end = NSMaxX(z);
        found = 1;
    };
    if ([NSThread isMainThread]) {
        measure();
    } else {
        dispatch_sync(dispatch_get_main_queue(), measure);
    }
    *titlebar = height;
    *controlsEnd = end;
    return found;
}
