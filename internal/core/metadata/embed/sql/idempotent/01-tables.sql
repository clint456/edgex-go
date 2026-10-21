--
-- Copyright (C) 2024 IOTech Ltd
--
-- SPDX-License-Identifier: Apache-2.0

-- core_metadata.node_service is used to store the node_service information
CREATE TABLE IF NOT EXISTS core_metadata.node_service (
    id UUID PRIMARY KEY,
    content JSONB NOT NULL
);

-- core_metadata.node_profile is used to store the node_profile information
CREATE TABLE IF NOT EXISTS core_metadata.node_profile (
    id UUID PRIMARY KEY,
    content JSONB NOT NULL
);

-- core_metadata.node is used to store the node information
CREATE TABLE IF NOT EXISTS core_metadata.node (
    id UUID PRIMARY KEY,
    content JSONB NOT NULL
);

-- core_metadata.provision_watcher is used to store the provision watcher information
CREATE TABLE IF NOT EXISTS core_metadata.provision_watcher (
    id UUID PRIMARY KEY,
    content JSONB NOT NULL
);
