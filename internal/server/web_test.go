package server

import "testing"

func TestWebURLVariableNormalizesPackageName(t *testing.T) {
	if got := WebURLVariable("los-hwrp"); got != "NWM_WEB_URL_LOS_HWRP" {
		t.Errorf("variable: %s", got)
	}
}
