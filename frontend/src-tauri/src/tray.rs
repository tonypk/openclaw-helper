use tauri::{
    menu::{MenuBuilder, MenuItemBuilder},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    Manager,
};

/// Create the system tray icon with right-click menu.
pub fn create_tray(app: &tauri::App) -> Result<(), Box<dyn std::error::Error>> {
    // Build tray menu
    let console_item =
        MenuItemBuilder::with_id("console", "Open OpenClaw Console / \u{6253}\u{5F00}\u{63A7}\u{5236}\u{53F0}")
            .build(app)?;
    let show_item = MenuItemBuilder::with_id("show", "Show Helper / \u{663E}\u{793A}\u{52A9}\u{624B}").build(app)?;
    let quit_item = MenuItemBuilder::with_id("quit", "Quit / \u{9000}\u{51FA}").build(app)?;

    let menu = MenuBuilder::new(app)
        .item(&console_item)
        .item(&show_item)
        .separator()
        .item(&quit_item)
        .build()?;

    let _tray = TrayIconBuilder::new()
        .tooltip("OpenClaw Helper")
        .menu(&menu)
        .on_menu_event(|app, event| match event.id().as_ref() {
            "console" => {
                let _ = open_url(app, "http://localhost:18789");
            }
            "show" => {
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.show();
                    let _ = window.set_focus();
                }
            }
            "quit" => {
                app.exit(0);
            }
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                let app = tray.app_handle();
                if let Some(window) = app.get_webview_window("main") {
                    let _ = window.show();
                    let _ = window.set_focus();
                }
            }
        })
        .build(app)?;

    Ok(())
}

/// Open a URL in the user's default browser using the shell plugin.
#[allow(deprecated)]
fn open_url(app: &tauri::AppHandle, url: &str) -> Result<(), Box<dyn std::error::Error>> {
    use tauri_plugin_shell::ShellExt;
    app.shell().open(url, None::<tauri_plugin_shell::open::Program>)?;
    Ok(())
}

/// Set up close-to-tray: hide window instead of quitting when X is clicked.
pub fn setup_close_to_tray(app: &tauri::App) {
    if let Some(window) = app.get_webview_window("main") {
        let win = window.clone();
        window.on_window_event(move |event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = win.hide();
            }
        });
    }
}
