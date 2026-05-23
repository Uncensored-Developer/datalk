CREATE UNIQUE INDEX users_single_owner_idx
    ON users (role)
    WHERE role = 'owner';
