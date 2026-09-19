package config

import "testing"

func TestProductionRejectsMocks(t *testing.T) {
	c := Config{AppEnv: "prod", RedisPrefix: "yingji:prod:", StorageDriver: "oss", GenProviderDefault: "newapi", NewAPIKey: "test", InspectProvider: "newapi", InspectModel: "vision", GenConcurrency: 1, DBMaxOpen: 5, DBMaxIdle: 1, JWTSecretMP: "012345678901234567890123456789012345", JWTSecretAdmin: "012345678901234567890123456789012345", SignSecret: "012345678901234567890123456789012345"}
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
	c.InspectProvider = "mock"
	if c.Validate() == nil {
		t.Fatal("生产环境不能用模拟照片质检")
	}
	c.InspectProvider = "newapi"
	c.GenProviderDefault = "mock"
	if c.Validate() == nil {
		t.Fatal("生产环境不能用模拟生图")
	}
}

func TestInspectNeedsModel(t *testing.T) {
	c := Config{AppEnv: "staging", RedisPrefix: "yingji:test:", StorageDriver: "oss", GenProviderDefault: "newapi", NewAPIKey: "test", InspectProvider: "newapi", GenConcurrency: 1, DBMaxOpen: 2}
	if c.Validate() == nil {
		t.Fatal("缺少 INSPECT_MODEL 时应拒绝启动")
	}
	c.InspectModel = "vision"
	if e := c.Validate(); e != nil {
		t.Fatal(e)
	}
}
