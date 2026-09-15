package config

import "testing"

func TestProductionRejectsMocks(t *testing.T) {
	c := Config{AppEnv: "prod", RedisPrefix: "yingji:prod:", StorageDriver: "oss", GenProviderDefault: "newapi", NewAPIKey: "test", FaceProvider: "disabled", GenConcurrency: 1, DBMaxOpen: 5, DBMaxIdle: 1, JWTSecretMP: "012345678901234567890123456789012345", JWTSecretAdmin: "012345678901234567890123456789012345", SignSecret: "012345678901234567890123456789012345"}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	c.FaceProvider = "mock"
	if c.Validate() == nil {
		t.Fatal("生产环境不能用模拟视觉校验")
	}
	c.FaceProvider = "disabled"
	c.GenProviderDefault = "mock"
	if c.Validate() == nil {
		t.Fatal("生产环境不能用模拟生图")
	}
}
