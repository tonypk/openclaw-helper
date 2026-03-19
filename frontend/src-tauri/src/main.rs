// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod commands;
mod helper_ipc;
mod tray;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
            commands::helper_rpc,
            commands::check_update,
            commands::report_telegram_fallback,
        ])
        .setup(|app| {
            // Launch Go helper sidecar
            helper_ipc::start_helper(app)?;

            // Set up system tray with menu
            tray::create_tray(app)?;

            // Close button hides to tray instead of quitting
            tray::setup_close_to_tray(app);

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
