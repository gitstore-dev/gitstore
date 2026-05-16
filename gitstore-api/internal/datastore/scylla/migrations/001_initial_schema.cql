-- Migration 001: initial schema.
-- Assumes the keyspace already exists (provisioned by the operator / compose init).
-- UUID fields use text for Go string compatibility. Price uses decimal;

CREATE TABLE IF NOT EXISTS products (
    id                 uuid       PRIMARY KEY,
    sku                text,
    title              text,
    price              decimal,
    currency           text,
    inventory_status   text,
    inventory_quantity int,
    category_id        text,
    collection_ids     set<text>,
    images             set<text>,
    metadata           map<text, text>,
    created_at         timestamp,
    updated_at         timestamp,
    body               text
);

CREATE TABLE IF NOT EXISTS categories (
    id                 uuid       PRIMARY KEY,
    name               text,
    slug               text,
    parent_id          text,
    display_order      int,
    created_at         timestamp,
    updated_at         timestamp,
    body               text
);

CREATE TABLE IF NOT EXISTS collections (
    id                 uuid       PRIMARY KEY,
    name               text,
    slug               text,
    display_order      int,
    product_ids        set<text>,
    created_at         timestamp,
    updated_at         timestamp,
    body               text
);
