# Error Page Templates

This directory contains HTML templates for the custom error pages served by the WASM plugin.

## Files

- **error-4xx.html** - Displayed for all 4xx client errors (400, 401, 403, 404, etc.)
- **error-5xx.html** - Displayed for all 5xx server errors (500, 502, 503, 504, etc.)

## Customizing Templates

These are standard HTML files that you can edit with any text editor. No Go programming knowledge is required!

### Quick Edits

**Change the text:**
- Edit any text between HTML tags (`<p>...</p>`, `<h1>...</h1>`, etc.)
- Modify button labels in the `<a>` tags

**Change colors:**
- Find the `<style>` section in the `<head>`
- Modify color values (e.g., `#f44336` for red, `#ff9800` for orange)
- Update `background: linear-gradient(...)` for background colors

**Change layout:**
- Modify the HTML structure inside `<div class="error-container">`
- Add or remove sections as needed

### After Making Changes

After editing the templates, you **must rebuild** the WASM plugin to embed the new HTML:

```bash
# Rebuild locally
make build

# Or rebuild Docker image
make build-docker
```

The templates are embedded into the compiled WASM binary, so changes won't take effect until you rebuild.

## Template Structure

Both templates follow the same structure:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Error Title</title>
    <style>
        /* CSS styles here */
    </style>
</head>
<body>
    <div class="error-container">
        <div class="error-icon">🔥</div>
        <h1>Error Heading</h1>
        <div class="error-code">ERROR CODE</div>
        <p class="error-message">Main message</p>
        <p class="error-description">Detailed description</p>
        <div class="action-buttons">
            <!-- Buttons here -->
        </div>
        <div class="footer">Footer text</div>
    </div>
</body>
</html>
```

## Styling Guide

### Color Schemes

**4xx Template (Client Errors):**
- Primary color: `#ff9800` (orange)
- Dark shade: `#f57c00`
- Background gradient: `#fef5e7` to `#fff3e0`

**5xx Template (Server Errors):**
- Primary color: `#f44336` (red)
- Dark shade: `#c62828`
- Background gradient: `#ffebee` to `#fce4ec`

### Responsive Design

Both templates are mobile-friendly and include responsive breakpoints:
- Desktop: Full layout with side-by-side buttons
- Mobile (<600px): Stacked layout with full-width buttons

## Adding Your Branding

You can easily add your company's branding:

1. **Logo**: Add an `<img>` tag inside the error-container
2. **Colors**: Replace the color values with your brand colors
3. **Fonts**: Add a `<link>` to Google Fonts or your custom font in the `<head>`
4. **Footer**: Update the footer text with your company name or support link

Example:
```html
<div class="error-container">
    <img src="data:image/svg+xml;base64,..." alt="Logo" style="width: 100px; margin: 0 auto 20px;">
    <!-- rest of the content -->
    <div class="footer">
        Need help? Contact <a href="mailto:support@yourcompany.com">support@yourcompany.com</a>
    </div>
</div>
```

## Tips

- **Keep it simple**: Users seeing error pages are already frustrated
- **Be helpful**: Provide clear next steps (go back, retry, contact support)
- **Test on mobile**: Many users will see errors on mobile devices
- **Use emoji sparingly**: They render differently across platforms
- **Self-contained**: Don't reference external CSS/JS files - they won't load if the backend is down

## Examples

### Adding a Support Email Link

```html
<div class="footer">
    Need help? Email us at <a href="mailto:support@example.com" style="color: #f44336;">support@example.com</a>
</div>
```

### Adding a Status Page Link

```html
<p class="error-description">
    Check our <a href="https://status.example.com" target="_blank" style="color: #f44336;">status page</a> 
    for real-time updates on system health.
</p>
```

### Customizing Button Actions

```html
<div class="action-buttons">
    <a href="javascript:history.back()" class="btn btn-secondary">Go Back</a>
    <a href="https://example.com/contact" class="btn btn-primary">Contact Support</a>
</div>
```

## Troubleshooting

**Q: I edited the template but nothing changed**
- A: You need to rebuild the WASM binary (`make build` or `make build-docker`)

**Q: Can I use external CSS or JavaScript?**
- A: No, keep everything inline. External resources may not load during errors.

**Q: Can I add images?**
- A: Yes, but use data URLs or SVG inline. External images won't load reliably.

**Q: How do I test my changes?**
- A: After rebuilding, trigger an error (e.g., `curl http://localhost:10000/error500`)

## Need Help?

If you need to make more advanced changes or add status-specific error pages, check the main README.md or the Go code in `main.go`.