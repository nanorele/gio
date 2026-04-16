package gpu

import "github.com/nanorele/gio/gpu/internal/driver"

type API = driver.API

type RenderTarget = driver.RenderTarget

type OpenGLRenderTarget = driver.OpenGLRenderTarget

type Direct3D11RenderTarget = driver.Direct3D11RenderTarget

type MetalRenderTarget = driver.MetalRenderTarget

type VulkanRenderTarget = driver.VulkanRenderTarget

type OpenGL = driver.OpenGL

type Direct3D11 = driver.Direct3D11

type Metal = driver.Metal

type Vulkan = driver.Vulkan

var ErrDeviceLost = driver.ErrDeviceLost
