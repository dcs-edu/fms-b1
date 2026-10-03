
-- 1. Drop the payments table first (since it depends on utility_prices)
DROP TABLE IF EXISTS payments;

-- 2. Drop the utility_prices table second (since it depends on utilities)
DROP TABLE IF EXISTS utility_prices;

-- 3. Drop the base utilities table last
DROP TABLE IF EXISTS utilities;
