package logger

import "testing"

func TestLogger(t *testing.T) {
	Logger.SetLevelOfOutput("DEBUG")
	Logger.Info("test info", "test info2")
	Logger.Infof("test info%s", "test info2")
	Logger.Infoln("test info", "test info2")

	Logger.Errorf("test error%s", "test error2")
	Logger.Errorln("test error", "test error2")
	Logger.Error("test error", "test error2")

	Logger.Warnf("test warn%s", "test warn2")
	Logger.Warnln("test warn", "test warn2")
	Logger.Warn("test warn", "test warn2")

	Logger.Debugf("test debug%s", "test debug2")
	Logger.Debugln("test debug", "test debug2")
	Logger.Debug("test debug", "test debug2")

	Logger.Fatalf("test fatal%s", "test fatal2")
	Logger.Fatalln("test fatal", "test fatal2")
	Logger.Fatal("test fatal", "test fatal2")
}
