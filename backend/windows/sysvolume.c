//go:build windows

#define COBJMACROS
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <objbase.h>
#include <mmdeviceapi.h>
#include <endpointvolume.h>
#include <math.h>

static const CLSID SUPERSONIC_CLSID_MMDeviceEnumerator = {
	0xbcde0395, 0xe52f, 0x467c, {0x8e, 0x3d, 0xc4, 0x57, 0x92, 0x91, 0x69, 0x2e}
};
static const IID SUPERSONIC_IID_IMMDeviceEnumerator = {
	0xa95664d2, 0x9614, 0x4f35, {0xa7, 0x46, 0xde, 0x8d, 0xb6, 0x36, 0x17, 0xe6}
};
static const IID SUPERSONIC_IID_IAudioEndpointVolume = {
	0x5cdf2c82, 0x841e, 0x4546, {0x97, 0x22, 0x0c, 0xf7, 0x40, 0x78, 0x22, 0x9a}
};

static IAudioEndpointVolume* get_default_endpoint_volume(int *com_inited) {
	HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
	if (hr == S_OK || hr == S_FALSE) {
		*com_inited = 1;
	} else {
		*com_inited = 0;
		if (hr != RPC_E_CHANGED_MODE) {
			return NULL;
		}
	}

	IMMDeviceEnumerator *enumerator = NULL;
	hr = CoCreateInstance(
		&SUPERSONIC_CLSID_MMDeviceEnumerator,
		NULL,
		CLSCTX_ALL,
		&SUPERSONIC_IID_IMMDeviceEnumerator,
		(void **)&enumerator
	);
	if (FAILED(hr) || enumerator == NULL) {
		if (*com_inited) {
			CoUninitialize();
			*com_inited = 0;
		}
		return NULL;
	}

	IMMDevice *device = NULL;
	hr = IMMDeviceEnumerator_GetDefaultAudioEndpoint(enumerator, eRender, eConsole, &device);
	IMMDeviceEnumerator_Release(enumerator);
	if (FAILED(hr) || device == NULL) {
		if (*com_inited) {
			CoUninitialize();
			*com_inited = 0;
		}
		return NULL;
	}

	IAudioEndpointVolume *epVol = NULL;
	hr = IMMDevice_Activate(device, &SUPERSONIC_IID_IAudioEndpointVolume, CLSCTX_ALL, NULL, (void **)&epVol);
	IMMDevice_Release(device);
	if (FAILED(hr) || epVol == NULL) {
		if (*com_inited) {
			CoUninitialize();
			*com_inited = 0;
		}
		return NULL;
	}

	return epVol;
}

static void release_endpoint_volume(IAudioEndpointVolume *epVol, int com_inited) {
	if (epVol != NULL) {
		IAudioEndpointVolume_Release(epVol);
	}
	if (com_inited) {
		CoUninitialize();
	}
}

int win_get_system_volume(int *out_vol, int *out_muted) {
	int com_inited = 0;
	IAudioEndpointVolume *epVol = get_default_endpoint_volume(&com_inited);
	if (epVol == NULL) {
		return -1;
	}

	float scalar = 0.0f;
	BOOL muted = FALSE;
	HRESULT hr1 = IAudioEndpointVolume_GetMasterVolumeLevelScalar(epVol, &scalar);
	HRESULT hr2 = IAudioEndpointVolume_GetMute(epVol, &muted);
	release_endpoint_volume(epVol, com_inited);

	if (FAILED(hr1) || FAILED(hr2)) {
		return -1;
	}

	int vol = (int)lroundf(scalar * 100.0f);
	if (vol < 0) vol = 0;
	if (vol > 100) vol = 100;
	*out_vol = vol;
	*out_muted = muted ? 1 : 0;
	return 0;
}

int win_set_system_volume(int vol) {
	if (vol < 0) vol = 0;
	if (vol > 100) vol = 100;

	int com_inited = 0;
	IAudioEndpointVolume *epVol = get_default_endpoint_volume(&com_inited);
	if (epVol == NULL) {
		return -1;
	}

	float scalar = (float)vol / 100.0f;
	HRESULT hr = IAudioEndpointVolume_SetMasterVolumeLevelScalar(epVol, scalar, NULL);
	if (SUCCEEDED(hr) && vol > 0) {
		IAudioEndpointVolume_SetMute(epVol, FALSE, NULL);
	}
	release_endpoint_volume(epVol, com_inited);
	return SUCCEEDED(hr) ? 0 : -1;
}

int win_set_system_mute(int muted) {
	int com_inited = 0;
	IAudioEndpointVolume *epVol = get_default_endpoint_volume(&com_inited);
	if (epVol == NULL) {
		return -1;
	}

	HRESULT hr = IAudioEndpointVolume_SetMute(epVol, muted ? TRUE : FALSE, NULL);
	release_endpoint_volume(epVol, com_inited);
	return SUCCEEDED(hr) ? 0 : -1;
}
