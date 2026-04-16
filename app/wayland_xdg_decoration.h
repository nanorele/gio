

#ifndef XDG_DECORATION_UNSTABLE_V1_CLIENT_PROTOCOL_H
#define XDG_DECORATION_UNSTABLE_V1_CLIENT_PROTOCOL_H

#include <stdint.h>
#include <stddef.h>
#include "wayland-client.h"

#ifdef  __cplusplus
extern "C" {
#endif


struct xdg_toplevel;
struct zxdg_decoration_manager_v1;
struct zxdg_toplevel_decoration_v1;

#ifndef ZXDG_DECORATION_MANAGER_V1_INTERFACE
#define ZXDG_DECORATION_MANAGER_V1_INTERFACE


extern const struct wl_interface zxdg_decoration_manager_v1_interface;
#endif
#ifndef ZXDG_TOPLEVEL_DECORATION_V1_INTERFACE
#define ZXDG_TOPLEVEL_DECORATION_V1_INTERFACE


extern const struct wl_interface zxdg_toplevel_decoration_v1_interface;
#endif

#define ZXDG_DECORATION_MANAGER_V1_DESTROY 0
#define ZXDG_DECORATION_MANAGER_V1_GET_TOPLEVEL_DECORATION 1



#define ZXDG_DECORATION_MANAGER_V1_DESTROY_SINCE_VERSION 1

#define ZXDG_DECORATION_MANAGER_V1_GET_TOPLEVEL_DECORATION_SINCE_VERSION 1


static inline void
zxdg_decoration_manager_v1_set_user_data(struct zxdg_decoration_manager_v1 *zxdg_decoration_manager_v1, void *user_data)
{
	wl_proxy_set_user_data((struct wl_proxy *) zxdg_decoration_manager_v1, user_data);
}


static inline void *
zxdg_decoration_manager_v1_get_user_data(struct zxdg_decoration_manager_v1 *zxdg_decoration_manager_v1)
{
	return wl_proxy_get_user_data((struct wl_proxy *) zxdg_decoration_manager_v1);
}

static inline uint32_t
zxdg_decoration_manager_v1_get_version(struct zxdg_decoration_manager_v1 *zxdg_decoration_manager_v1)
{
	return wl_proxy_get_version((struct wl_proxy *) zxdg_decoration_manager_v1);
}


static inline void
zxdg_decoration_manager_v1_destroy(struct zxdg_decoration_manager_v1 *zxdg_decoration_manager_v1)
{
	wl_proxy_marshal((struct wl_proxy *) zxdg_decoration_manager_v1,
			 ZXDG_DECORATION_MANAGER_V1_DESTROY);

	wl_proxy_destroy((struct wl_proxy *) zxdg_decoration_manager_v1);
}


static inline struct zxdg_toplevel_decoration_v1 *
zxdg_decoration_manager_v1_get_toplevel_decoration(struct zxdg_decoration_manager_v1 *zxdg_decoration_manager_v1, struct xdg_toplevel *toplevel)
{
	struct wl_proxy *id;

	id = wl_proxy_marshal_constructor((struct wl_proxy *) zxdg_decoration_manager_v1,
			 ZXDG_DECORATION_MANAGER_V1_GET_TOPLEVEL_DECORATION, &zxdg_toplevel_decoration_v1_interface, NULL, toplevel);

	return (struct zxdg_toplevel_decoration_v1 *) id;
}

#ifndef ZXDG_TOPLEVEL_DECORATION_V1_ERROR_ENUM
#define ZXDG_TOPLEVEL_DECORATION_V1_ERROR_ENUM
enum zxdg_toplevel_decoration_v1_error {
	
	ZXDG_TOPLEVEL_DECORATION_V1_ERROR_UNCONFIGURED_BUFFER = 0,
	
	ZXDG_TOPLEVEL_DECORATION_V1_ERROR_ALREADY_CONSTRUCTED = 1,
	
	ZXDG_TOPLEVEL_DECORATION_V1_ERROR_ORPHANED = 2,
};
#endif 

#ifndef ZXDG_TOPLEVEL_DECORATION_V1_MODE_ENUM
#define ZXDG_TOPLEVEL_DECORATION_V1_MODE_ENUM

enum zxdg_toplevel_decoration_v1_mode {
	
	ZXDG_TOPLEVEL_DECORATION_V1_MODE_CLIENT_SIDE = 1,
	
	ZXDG_TOPLEVEL_DECORATION_V1_MODE_SERVER_SIDE = 2,
};
#endif 


struct zxdg_toplevel_decoration_v1_listener {
	
	void (*configure)(void *data,
			  struct zxdg_toplevel_decoration_v1 *zxdg_toplevel_decoration_v1,
			  uint32_t mode);
};


static inline int
zxdg_toplevel_decoration_v1_add_listener(struct zxdg_toplevel_decoration_v1 *zxdg_toplevel_decoration_v1,
					 const struct zxdg_toplevel_decoration_v1_listener *listener, void *data)
{
	return wl_proxy_add_listener((struct wl_proxy *) zxdg_toplevel_decoration_v1,
				     (void (**)(void)) listener, data);
}

#define ZXDG_TOPLEVEL_DECORATION_V1_DESTROY 0
#define ZXDG_TOPLEVEL_DECORATION_V1_SET_MODE 1
#define ZXDG_TOPLEVEL_DECORATION_V1_UNSET_MODE 2


#define ZXDG_TOPLEVEL_DECORATION_V1_CONFIGURE_SINCE_VERSION 1


#define ZXDG_TOPLEVEL_DECORATION_V1_DESTROY_SINCE_VERSION 1

#define ZXDG_TOPLEVEL_DECORATION_V1_SET_MODE_SINCE_VERSION 1

#define ZXDG_TOPLEVEL_DECORATION_V1_UNSET_MODE_SINCE_VERSION 1


static inline void
zxdg_toplevel_decoration_v1_set_user_data(struct zxdg_toplevel_decoration_v1 *zxdg_toplevel_decoration_v1, void *user_data)
{
	wl_proxy_set_user_data((struct wl_proxy *) zxdg_toplevel_decoration_v1, user_data);
}


static inline void *
zxdg_toplevel_decoration_v1_get_user_data(struct zxdg_toplevel_decoration_v1 *zxdg_toplevel_decoration_v1)
{
	return wl_proxy_get_user_data((struct wl_proxy *) zxdg_toplevel_decoration_v1);
}

static inline uint32_t
zxdg_toplevel_decoration_v1_get_version(struct zxdg_toplevel_decoration_v1 *zxdg_toplevel_decoration_v1)
{
	return wl_proxy_get_version((struct wl_proxy *) zxdg_toplevel_decoration_v1);
}


static inline void
zxdg_toplevel_decoration_v1_destroy(struct zxdg_toplevel_decoration_v1 *zxdg_toplevel_decoration_v1)
{
	wl_proxy_marshal((struct wl_proxy *) zxdg_toplevel_decoration_v1,
			 ZXDG_TOPLEVEL_DECORATION_V1_DESTROY);

	wl_proxy_destroy((struct wl_proxy *) zxdg_toplevel_decoration_v1);
}


static inline void
zxdg_toplevel_decoration_v1_set_mode(struct zxdg_toplevel_decoration_v1 *zxdg_toplevel_decoration_v1, uint32_t mode)
{
	wl_proxy_marshal((struct wl_proxy *) zxdg_toplevel_decoration_v1,
			 ZXDG_TOPLEVEL_DECORATION_V1_SET_MODE, mode);
}


static inline void
zxdg_toplevel_decoration_v1_unset_mode(struct zxdg_toplevel_decoration_v1 *zxdg_toplevel_decoration_v1)
{
	wl_proxy_marshal((struct wl_proxy *) zxdg_toplevel_decoration_v1,
			 ZXDG_TOPLEVEL_DECORATION_V1_UNSET_MODE);
}

#ifdef  __cplusplus
}
#endif

#endif
