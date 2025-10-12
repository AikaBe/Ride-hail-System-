begin;

-- Drop dependent tables first
drop table if exists location_history cascade;
drop table if exists driver_sessions cascade;
drop table if exists drivers cascade;

-- Drop enumeration tables
drop table if exists "driver_status" cascade;

commit;
