package id

import "github.com/bwmarrin/snowflake"

func NextId(svr int64) (int64, error) {
	node, err := snowflake.NewNode(svr)
	if err != nil {
		return 0, err
	}

	// Generate a snowflake ID.
	id := node.Generate()
	return id.Int64(), nil
}
