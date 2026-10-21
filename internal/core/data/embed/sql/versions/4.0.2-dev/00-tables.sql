--
-- Copyright (C) 2025 IOTech Ltd
--
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE core_data.node_info ADD COLUMN IF NOT EXISTS mark_deleted BOOLEAN DEFAULT false;

-- create index on reading(event_id) to enhance the query performance
CREATE INDEX IF NOT EXISTS idx_reading_event_id ON core_data.reading(event_id);

ALTER TABLE core_data.reading ADD COLUMN IF NOT EXISTS numeric_value NUMERIC;

-- create index on reading(node_info_id) to enhance the performance of queries that join reading with node_info on node_info_id
CREATE INDEX IF NOT EXISTS idx_reading_node_info_id ON core_data.reading(node_info_id);

-- create index on event(node_info_id) to enhance the performance of queries that join event with node_info on node_info_id
CREATE INDEX IF NOT EXISTS idx_event_node_info_id ON core_data.event(node_info_id);

-- -- create index on reading and node_info to enhance the performance of queries that join reading with node_info and specified node and resource
CREATE INDEX IF NOT EXISTS idx_reading_node_info_origin ON core_data.reading(node_info_id, origin DESC);
CREATE INDEX IF NOT EXISTS idx_node_info_node_resource ON core_data.node_info(mark_deleted, nodename, resourcename);
