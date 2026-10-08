-- Preserve the initial protocol/key prototype table if it exists. Its data cannot
-- be translated losslessly into the point model, so retain it for manual migration.
DO $$
BEGIN
    IF to_regclass('support_mappings.mapping') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema = 'support_mappings'
             AND table_name = 'mapping'
             AND column_name = 'north_resource_name'
       ) THEN
        ALTER TABLE support_mappings.mapping RENAME TO mapping_legacy;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS support_mappings.mapping (
    id UUID PRIMARY KEY,
    north_device_name TEXT NOT NULL,
    north_profile_name TEXT NOT NULL,
    north_resource_name TEXT NOT NULL,
    south_device_name TEXT NOT NULL,
    south_profile_name TEXT NOT NULL,
    south_resource_name TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created BIGINT NOT NULL,
    modified BIGINT NOT NULL,
    UNIQUE (north_device_name, north_profile_name, north_resource_name),
    UNIQUE (south_device_name, south_profile_name, south_resource_name)
);

CREATE INDEX IF NOT EXISTS mapping_north_device_name_idx ON support_mappings.mapping (north_device_name);
CREATE INDEX IF NOT EXISTS mapping_south_device_name_idx ON support_mappings.mapping (south_device_name);
CREATE INDEX IF NOT EXISTS mapping_north_profile_name_idx ON support_mappings.mapping (north_profile_name);
CREATE INDEX IF NOT EXISTS mapping_south_profile_name_idx ON support_mappings.mapping (south_profile_name);
