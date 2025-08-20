-- create "users" table
CREATE TABLE users (
	id bigserial NOT NULL,
	name varchar(255) NOT NULL,
	email varchar(255) NOT NULL,
	created_at timestamp NOT NULL,
	PRIMARY KEY (id),
	UNIQUE (email)
);

-- create "posts" table
CREATE TABLE posts (
	id bigserial NOT NULL,
	title varchar(255) NOT NULL,
	body text NOT NULL,
	created_at timestamp NOT NULL,
	user_posts bigint NULL,
	PRIMARY KEY (id),
	CONSTRAINT posts_users_posts FOREIGN KEY (user_posts) REFERENCES users (id) ON UPDATE NO ACTION ON DELETE SET NULL
);

CREATE INDEX posts_users_posts ON posts (user_posts);
