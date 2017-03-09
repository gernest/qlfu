begin transaction;
   create table projects (
    id         int64,
    user_id    int64,
    name       string,
    created_at time,
    updated_at time);
   create table sessions (
    data       blob,
    id         int64,
    key        string,
    created_on time,
    updated_on time,
    expires_on time);
   create table tasks (
    done       bool,
    id         int64,
    user_id    int64,
    project_id int64,
    uuid       string,
    created_at time,
    updated_at time);
   create table users (
    password   blob,
    id         int64,
    username   string,
    email      string,
    created_at time,
    updated_at time);
commit;
