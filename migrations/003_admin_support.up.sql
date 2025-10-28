begin;

-- Ensure all required tables exist for admin functionality
-- This migration ensures the database has all necessary tables and indexes

-- Create index for better ride status queries
CREATE INDEX IF NOT EXISTS idx_rides_status_created ON rides(status, created_at);

-- Create index for driver status queries
CREATE INDEX IF NOT EXISTS idx_drivers_status_updated ON drivers(status, updated_at);

-- Create index for coordinates by entity
CREATE INDEX IF NOT EXISTS idx_coordinates_entity_current ON coordinates(entity_id, entity_type, is_current);

-- Create index for ride events by timestamp
CREATE INDEX IF NOT EXISTS idx_ride_events_created ON ride_events(created_at);

-- Create view for admin dashboard (optional but helpful)
CREATE OR REPLACE VIEW admin_dashboard_stats AS
SELECT
    (SELECT COUNT(*) FROM rides WHERE status IN ('REQUESTED', 'MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')) as active_rides,
    (SELECT COUNT(*) FROM drivers WHERE status = 'AVAILABLE') as available_drivers,
    (SELECT COUNT(*) FROM drivers WHERE status IN ('BUSY', 'EN_ROUTE')) as busy_drivers,
    (SELECT COUNT(*) FROM rides WHERE DATE(created_at) = CURRENT_DATE) as today_rides,
    (SELECT COALESCE(SUM(final_fare), 0) FROM rides WHERE DATE(created_at) = CURRENT_DATE AND status = 'COMPLETED') as today_revenue;

commit;