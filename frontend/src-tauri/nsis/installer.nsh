; Kill running OpenClaw Helper processes before installing/upgrading.
; This prevents "file in use" errors when overwriting och-helper.exe.

!macro NSIS_HOOK_PREINSTALL
  ; Kill the Go sidecar process
  nsExec::ExecToLog 'taskkill /F /IM "och-helper-x86_64-pc-windows-msvc.exe" 2>nul'
  nsExec::ExecToLog 'taskkill /F /IM "och-helper.exe" 2>nul'
  ; Kill the main Tauri app process
  nsExec::ExecToLog 'taskkill /F /IM "OpenClaw Helper.exe" 2>nul'
  ; Brief pause to let OS release file locks
  Sleep 1000
!macroend
