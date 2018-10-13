create table if not exists visit_record (
    username VARCHAR(64) NOT NULL,
    host VARCHAR(1024) NOT NULL,
    user_from VARCHAR(32) NOT NULL,
    start_time DATE,
    end_time DATE,
    read_cnt BIGINT,
    write_cnt BIGINT,
    connect_cost int
)