--
-- Copyright (C) 2026 IOTech Ltd
--
-- SPDX-License-Identifier: Apache-2.0

-- idx_node_content_gin is a GIN index on the node content column to accelerate JSONB containment queries with '@>' operators,
-- such as lookups by ProfileName and ServiceName
CREATE INDEX IF NOT EXISTS idx_node_content_gin ON core_metadata.node USING GIN (content jsonb_path_ops);
