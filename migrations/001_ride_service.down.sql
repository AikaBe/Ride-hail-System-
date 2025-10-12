begin;

-- Drop event sourcing tables first (depend on rides)
drop table if exists ride_events cascade;
drop table if exists "ride_event_type" cascade;

-- Drop main rides-related tables
drop table if exists rides cascade;
drop table if exists coordinates cascade;

-- Drop enumeration tables
drop table if exists "vehicle_type" cascade;
drop table if exists "ride_status" cascade;

-- Drop user-related tables
drop table if exists users cascade;
drop table if exists "user_status" cascade;
drop table if exists "roles" cascade;

commit;
