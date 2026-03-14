// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod commands;
mod helper_ipc;
mod tray;

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![commands::helper_rpc])
        .setup(|app| {
            // Launch Go helper sidecar
            helper_ipc::start_helper(app.handle().clone())?;

            // Set up system tray
            tray::create_tray(app)?;

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
