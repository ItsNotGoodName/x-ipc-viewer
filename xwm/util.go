package xwm

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func CreateXSubWindow(x *xgb.Conn, root xproto.Window) (xproto.Window, error) {
	// Generate X window id
	wid, err := xproto.NewWindowId(x)
	if err != nil {
		return 0, err
	}

	// Create X window in root
	if xproto.CreateWindowChecked(x, xproto.WindowClassCopyFromParent,
		wid, root,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOutput, xproto.WindowClassCopyFromParent, 0, []uint32{}).Check(); err != nil {
		return 0, err
	}

	// Show X window
	if err = xproto.MapWindowChecked(x, wid).Check(); err != nil {
		xproto.DestroyWindow(x, wid)
		return 0, err
	}

	return wid, nil
}
