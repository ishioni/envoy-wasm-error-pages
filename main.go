// Copyright 2020-2024 Tetrate
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	_ "embed"

	"envoy-wasm-error-pages/internal/config"
	"envoy-wasm-error-pages/internal/errorpages"
	"envoy-wasm-error-pages/templates"

	"github.com/proxy-wasm/proxy-wasm-go-sdk/proxywasm"
	"github.com/proxy-wasm/proxy-wasm-go-sdk/proxywasm/types"
)

// version is set at compile time via ldflags
var version = "dev"

//go:embed config.yaml
var configYAML []byte

// Global handlers and config initialized at plugin start
var (
	errorPageHandler *errorpages.Handler
	pluginConfig     *config.Config
)

func main() {}

func init() {
	proxywasm.SetVMContext(&vmContext{})
}

// vmContext implements types.VMContext.
type vmContext struct {
	types.DefaultVMContext
}

// NewPluginContext implements types.VMContext.
func (*vmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{}
}

// pluginContext implements types.PluginContext.
type pluginContext struct {
	types.DefaultPluginContext
}

// NewHttpContext implements types.PluginContext.
func (ctx *pluginContext) NewHttpContext(contextID uint32) types.HttpContext {
	return &httpContext{}
}

// OnPluginStart implements types.PluginContext.
func (ctx *pluginContext) OnPluginStart(pluginConfigurationSize int) types.OnPluginStartStatus {
	proxywasm.LogInfo("WASM Error Pages Plugin initialized (version: " + version + ")")

	// Parse configuration
	var err error
	pluginConfig, err = config.Parse(configYAML)
	if err != nil {
		proxywasm.LogCriticalf("Failed to parse config.yaml: %v", err)
		return types.OnPluginStartStatusFailed
	}

	// Select template based on theme configuration
	templateBytes, err := templates.GetTemplate(pluginConfig.Theme)
	if err != nil {
		proxywasm.LogWarnf("Theme '%s' not found, falling back to 'app-down'", pluginConfig.Theme)
		templateBytes, err = templates.GetTemplate("app-down")
		if err != nil {
			proxywasm.LogCriticalf("Failed to load fallback template: %v", err)
			return types.OnPluginStartStatusFailed
		}
		pluginConfig.Theme = "app-down"
	}

	// Initialize error page handler with selected template
	errorPageHandler, err = errorpages.NewWithTemplate(templateBytes, version)
	if err != nil {
		proxywasm.LogCriticalf("Failed to parse template: %v", err)
		return types.OnPluginStartStatusFailed
	}

	proxywasm.LogInfof("Error page template loaded: theme=%s, show_details=%v", pluginConfig.Theme, pluginConfig.ShowDetails)
	return types.OnPluginStartStatusOK
}

// httpContext implements types.HttpContext.
type httpContext struct {
	types.DefaultHttpContext

	shouldReplaceBody bool
	statusCode        string
	// Request data for template rendering
	host         string
	originalURI  string
	forwardedFor string
	requestID    string
}

// OnHttpRequestHeaders implements types.HttpContext.
func (ctx *httpContext) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	// Capture request data for error page rendering
	if host, err := proxywasm.GetHttpRequestHeader(":authority"); err == nil {
		ctx.host = host
	} else if host, err := proxywasm.GetHttpRequestHeader("host"); err == nil {
		ctx.host = host
	}

	if path, err := proxywasm.GetHttpRequestHeader(":path"); err == nil {
		ctx.originalURI = path
	}

	if xff, err := proxywasm.GetHttpRequestHeader("x-forwarded-for"); err == nil {
		ctx.forwardedFor = xff
	}

	if reqID, err := proxywasm.GetHttpRequestHeader("x-request-id"); err == nil {
		ctx.requestID = reqID
	}

	return types.ActionContinue
}

// OnHttpResponseHeaders implements types.HttpContext.
func (ctx *httpContext) OnHttpResponseHeaders(numHeaders int, endOfStream bool) types.Action {
	status, err := proxywasm.GetHttpResponseHeader(":status")
	if err != nil {
		proxywasm.LogWarnf("failed to get status code: %v", err)
		return types.ActionContinue
	}

	proxywasm.LogDebugf("response status code: %s", status)

	// Check if this is a 4xx or 5xx error
	if errorpages.IsErrorStatus(status) {
		ctx.shouldReplaceBody = true
		ctx.statusCode = status
		proxywasm.LogInfof("intercepting error response: %s", status)

		// Remove headers that could conflict with our custom error page
		proxywasm.RemoveHttpResponseHeader("content-length")
		proxywasm.RemoveHttpResponseHeader("content-encoding")
		proxywasm.RemoveHttpResponseHeader("content-type")

		// Set content type for our HTML error page
		proxywasm.AddHttpResponseHeader("content-type", "text/html; charset=utf-8")
	}

	return types.ActionContinue
}

// OnHttpResponseBody implements types.HttpContext.
func (ctx *httpContext) OnHttpResponseBody(bodySize int, endOfStream bool) types.Action {
	if !ctx.shouldReplaceBody {
		return types.ActionContinue
	}

	if !endOfStream {
		// Wait until we see the entire body to replace.
		return types.ActionPause
	}

	// Parse status code to int
	statusCode := 0
	for i := 0; i < len(ctx.statusCode); i++ {
		if ctx.statusCode[i] >= '0' && ctx.statusCode[i] <= '9' {
			statusCode = statusCode*10 + int(ctx.statusCode[i]-'0')
		}
	}

	// Build template data
	templateData := &errorpages.TemplateData{
		Code:         statusCode,
		ShowDetails:  pluginConfig.ShowDetails,
		Host:         ctx.host,
		OriginalURI:  ctx.originalURI,
		ForwardedFor: ctx.forwardedFor,
		RequestID:    ctx.requestID,
	}

	// Render the error page with template
	errorPage, err := errorPageHandler.RenderErrorPage(templateData)
	if err != nil {
		proxywasm.LogErrorf("failed to render error page: %v", err)
		return types.ActionContinue
	}

	// Replace the response body with our custom error page
	err = proxywasm.ReplaceHttpResponseBody(errorPage)
	if err != nil {
		proxywasm.LogErrorf("failed to replace response body: %v", err)
		return types.ActionContinue
	}

	proxywasm.LogDebugf("replaced error page for status: %s", ctx.statusCode)
	return types.ActionContinue
}
