package docker

//import (
//	"context"
//
//	"github.com/docker/docker/api/types"
//)
//
//func ListConfigs() ([]types.Config, error) {
//	c, err := GetClient()
//	if err != nil {
//		return nil, err
//	}
//	return c.ConfigList(context.Background(), types.ConfigListOptions{})
//}
//
//func InspectConfig(id string) (types.Config, error) {
//	c, err := GetClient()
//	if err != nil {
//		return types.Config{}, err
//	}
//	cfg, _, err := c.ConfigInspectWithRaw(context.Background(), id)
//	return cfg, err
//}
//
//func RemoveConfig(id string) error {
//	c, err := GetClient()
//	if err != nil {
//		return err
//	}
//	return c.ConfigRemove(context.Background(), id)
//}
