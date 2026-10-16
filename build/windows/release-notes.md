Connect 0.1.7 · Windows 10/11 x64

Update and rollback actions now start immediately when clicked. Removed browser confirmation dialogs from all component and application updates.

Component updates take effect on the next connection. Application updates close Connect and launch the Windows installer; Windows may still request administrator permission. Disable the Chrome extension while replacing the application, then re-enable it. Profiles are preserved.

The update interface test rejects confirmation dialogs and checks installation and rollback. The release workflow also runs core, native helper, extension and interface checks.

Mist remains experimental. No connectivity fixes are included. Executables are unsigned; no server credentials are bundled.
