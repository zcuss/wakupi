//go:build windows

package desktop

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// volumeScript is one PowerShell invocation that uses the COM
// IAudioEndpointVolume API to either read the current scalar level or set
// it to an absolute percentage in [0, 100]. We compile the C# block once
// per process — Add-Type is idempotent within the same PowerShell run.
//
// The C# uses raw P/Invoke + COM (no WMI, no SendKeys) so:
//
//   - !volume 50 no longer spawns 50+50=100 keystroke events.
//   - We get the actual current volume, not a hardcoded "50".
//   - The volume changes in one shot instead of relying on a key repeat race
//     against the user's focus window.
const volumeScriptTemplate = `
$src = @"
using System;
using System.Runtime.InteropServices;

[ComImport, Guid("a95664d2-9614-4f35-a746-de8db636c292"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
internal interface IMMDeviceEnumerator {
    [PreserveSig] int EnumAudioEndpoints(int dataFlow, int stateMask, out IntPtr ppDevices);
    [PreserveSig] int GetDefaultAudioEndpoint(int dataFlow, int role, out IMMDevice ppDevice);
}

[ComImport, Guid("d666063f-1587-4e43-81f1-a94870dab787"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
internal interface IMMDevice {
    [PreserveSig] int Activate(ref Guid iid, int dwClsCtx, IntPtr pActivationParams, out IntPtr ppInterface);
}

[ComImport, Guid("5cdf2c82-841e-4546-9722-0cf74078229a"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
internal interface IAudioEndpointVolume {
    int _pad1(); int _pad2(); int _pad3(); int _pad4(); int _pad5(); int _pad6();
    [PreserveSig] int GetMasterVolumeLevelScalar(out float level);
    [PreserveSig] int SetMasterVolumeLevelScalar(float level, ref Guid pguidEventContext);
    int _pad7(); int _pad8(); int _pad9(); int _pad10(); int _pad11();
}

public static class AudioHelper {
    private static readonly Guid CLSID_MMDeviceEnumerator = new Guid("bcde0395-e52f-467c-8e3d-c4579291692e");
    private static readonly Guid IID_IMMDeviceEnumerator = new Guid("a95664d2-9614-4f35-a746-de8db636c292");
    private static readonly Guid IID_IAudioEndpointVolume = new Guid("5cdf2c82-841e-4546-9722-0cf74078229a");

    public static int GetMasterPct() {
        try {
            var enumerator = (IMMDeviceEnumerator)Activator.CreateInstance(Type.GetTypeFromCLSID(CLSID_MMDeviceEnumerator, true));
            IMMDevice device;
            enumerator.GetDefaultAudioEndpoint(0, 1, out device);
            if (device == null) return -1;
            IntPtr pVol;
            var iid = IID_IAudioEndpointVolume;
            device.Activate(ref iid, 0, IntPtr.Zero, out pVol);
            if (pVol == IntPtr.Zero) return -1;
            var vol = (IAudioEndpointVolume)Marshal.GetTypedObjectForIUnknown(pVol, typeof(IAudioEndpointVolume));
            Marshal.Release(pVol);
            float level = 0f;
            vol.GetMasterVolumeLevelScalar(out level);
            return (int)Math.Round(level * 100f);
        } catch { return -1; }
    }

    public static bool SetMasterPct(int pct) {
        try {
            var enumerator = (IMMDeviceEnumerator)Activator.CreateInstance(Type.GetTypeFromCLSID(CLSID_MMDeviceEnumerator, true));
            IMMDevice device;
            enumerator.GetDefaultAudioEndpoint(0, 1, out device);
            if (device == null) return false;
            IntPtr pVol;
            var iid = IID_IAudioEndpointVolume;
            device.Activate(ref iid, 0, IntPtr.Zero, out pVol);
            if (pVol == IntPtr.Zero) return false;
            var vol = (IAudioEndpointVolume)Marshal.GetTypedObjectForIUnknown(pVol, typeof(IAudioEndpointVolume));
            Marshal.Release(pVol);
            float level = Math.Max(0f, Math.Min(1f, pct / 100f));
            var ctx = Guid.Empty;
            return vol.SetMasterVolumeLevelScalar(level, ref ctx) == 0;
        } catch { return false; }
    }
}
"@
Add-Type -TypeDefinition $src -Language CSharp
$mode = $args[0]
if ($mode -eq 'get') {
    [AudioHelper]::GetMasterPct()
} else {
    $pct = [int]$args[1]
    if ([AudioHelper]::SetMasterPct($pct)) { 'ok' } else { 'err' }
}
`

// runVolumeScript runs the volumeScriptTemplate with the given mode/pct and
// returns the trimmed stdout. Stderr is intentionally captured too so a
// hung PowerShell doesn't block the caller.
func runVolumeScript(mode, pct string) (string, error) {
	ps := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", volumeScriptTemplate, mode, pct)
	out, err := ps.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// GetVolume returns the current master volume as a 0..100 integer.
// Replaces the prior hardcoded "50" stub.
func (w *windowsController) GetVolume() (int, error) {
	out, err := runVolumeScript("get", "")
	if err != nil {
		return 0, fmt.Errorf("powershell get volume: %v: %s", err, out)
	}
	v, err := strconv.Atoi(out)
	if err != nil {
		return 0, fmt.Errorf("parse volume %q: %v", out, err)
	}
	return v, nil
}

// SetVolume sets the master volume to pct in [0, 100] in a single
// PowerShell call. The old code spammed 100+ SendKeys events, which is
// noisy and races with the user's focus window.
func (w *windowsController) SetVolume(pct int) error {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	out, err := runVolumeScript("set", strconv.Itoa(pct))
	if err != nil {
		return fmt.Errorf("powershell set volume: %v: %s", err, out)
	}
	if out != "ok" {
		return fmt.Errorf("set volume failed: %s", out)
	}
	return nil
}
