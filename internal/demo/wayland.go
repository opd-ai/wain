package demo

import (
	"fmt"
	"os"

	"github.com/opd-ai/wain/internal/wayland/client"
	"github.com/opd-ai/wain/internal/wayland/dmabuf"
	"github.com/opd-ai/wain/internal/wayland/shm"
	"github.com/opd-ai/wain/internal/wayland/xdg"
)

// WaylandContext holds common Wayland objects used across demos.
type WaylandContext struct {
	Conn       *client.Connection
	Compositor *client.Compositor
	SHM        *shm.SHM
	WmBase     *xdg.WmBase
}

// ConnectToWayland establishes a connection to the Wayland compositor.
// Returns the connection or an error if connection fails.
func ConnectToWayland() (*client.Connection, error) {
	display := os.Getenv("WAYLAND_DISPLAY")
	if display == "" {
		display = "wayland-0"
	}

	conn, err := client.Connect(display)
	if err != nil {
		return nil, fmt.Errorf("connect to Wayland: %w", err)
	}
	return conn, nil
}

// SetupWaylandGlobals discovers and binds to required Wayland global objects.
// This function handles the common pattern of:
//  1. Getting the registry
//  2. Finding and binding to wl_compositor
//  3. Finding and binding to wl_shm
//  4. Finding and binding to xdg_wm_base
func SetupWaylandGlobals(conn *client.Connection) (*WaylandContext, error) {
	registry, err := conn.Display().GetRegistry()
	if err != nil {
		return nil, fmt.Errorf("get registry: %w", err)
	}

	compositorGlobal := registry.FindGlobal("wl_compositor")
	if compositorGlobal == nil {
		return nil, fmt.Errorf("wl_compositor not found")
	}
	compositor, err := registry.BindCompositor(compositorGlobal)
	if err != nil {
		return nil, fmt.Errorf("bind compositor: %w", err)
	}

	shmObj, err := bindSHMGlobal(conn, registry)
	if err != nil {
		return nil, err
	}

	wmBase, err := bindXdgWmBase(conn, registry)
	if err != nil {
		return nil, err
	}

	return &WaylandContext{
		Conn:       conn,
		Compositor: compositor,
		SHM:        shmObj,
		WmBase:     wmBase,
	}, nil
}

// BindSHMGlobal finds the wl_shm global, binds it, and registers the object
// with the connection.
func BindSHMGlobal(conn *client.Connection, registry *client.Registry) (*shm.SHM, error) {
	shmGlobal := registry.FindGlobal("wl_shm")
	if shmGlobal == nil {
		return nil, fmt.Errorf("wl_shm not found")
	}
	shmID, err := registry.Bind(shmGlobal.Name, "wl_shm", shmGlobal.Version)
	if err != nil {
		return nil, fmt.Errorf("bind shm: %w", err)
	}
	shmObj := shm.NewSHM(conn, shmID)
	conn.RegisterObject(shmObj)
	return shmObj, nil
}

// bindSHMGlobal is an unexported alias kept for internal use within this package.
func bindSHMGlobal(conn *client.Connection, registry *client.Registry) (*shm.SHM, error) {
	return BindSHMGlobal(conn, registry)
}

// BindXdgWmBase finds the xdg_wm_base global, binds it, and registers the
// object with the connection.
func BindXdgWmBase(conn *client.Connection, registry *client.Registry) (*xdg.WmBase, error) {
	xdgGlobal := registry.FindGlobal("xdg_wm_base")
	if xdgGlobal == nil {
		return nil, fmt.Errorf("xdg_wm_base not found")
	}
	wmBaseID, _, err := registry.BindXdgWmBase(xdgGlobal)
	if err != nil {
		return nil, fmt.Errorf("bind xdg_wm_base: %w", err)
	}
	wmBase := xdg.NewWmBase(conn, wmBaseID, xdgGlobal.Version)
	conn.RegisterObject(wmBase)
	return wmBase, nil
}

// bindXdgWmBase is an unexported alias kept for internal use within this package.
func bindXdgWmBase(conn *client.Connection, registry *client.Registry) (*xdg.WmBase, error) {
	return BindXdgWmBase(conn, registry)
}

// BindDmabuf finds the zwp_linux_dmabuf_v1 global, binds it, and registers
// the object with the connection.
func BindDmabuf(conn *client.Connection, registry *client.Registry) (*dmabuf.Dmabuf, error) {
	g := registry.FindGlobal("zwp_linux_dmabuf_v1")
	if g == nil {
		return nil, fmt.Errorf("zwp_linux_dmabuf_v1 not supported by compositor")
	}
	id, err := registry.BindDmabuf(g)
	if err != nil {
		return nil, fmt.Errorf("failed to bind dmabuf: %w", err)
	}
	obj := dmabuf.NewDmabuf(conn, id)
	conn.RegisterObject(obj)
	return obj, nil
}

// CreateXdgWindow creates an XDG toplevel window with the specified title.
// This extracts the common pattern of creating an xdg_surface and toplevel.
func CreateXdgWindow(conn *client.Connection, wmBase *xdg.WmBase, surface *client.Surface, title string) (*xdg.Surface, *xdg.Toplevel, error) {
	xdgSurface, err := wmBase.GetXdgSurface(surface.ID())
	if err != nil {
		return nil, nil, fmt.Errorf("get xdg_surface: %w", err)
	}

	toplevel, err := xdgSurface.GetToplevel()
	if err != nil {
		return nil, nil, fmt.Errorf("get toplevel: %w", err)
	}

	if title != "" {
		if err := toplevel.SetTitle(title); err != nil {
			return nil, nil, fmt.Errorf("set title: %w", err)
		}
	}

	return xdgSurface, toplevel, nil
}
