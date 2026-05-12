-- Migration 001: initial schema
-- Assumes the keyspace already exists (provisioned by the operator / compose init).
-- UUID fields use text for Go string compatibility; price uses double.

CREATE TABLE IF NOT EXISTS products (
    id                 text       PRIMARY KEY,
    sku                text,
    title              text,
    price              double,
    currency           text,
    inventory_status   text,
    inventory_quantity int,
    category_id        text,
    collection_ids     list<text>,
    images             list<text>,
    metadata           map<text, text>,
    created_at         timestamp,
    updated_at         timestamp,
    body               text
);

CREATE TABLE IF NOT EXISTS categories (
    id            text  PRIMARY KEY,
    name          text,
    slug          text,
    parent_id     text,
    display_order int,
    created_at    timestamp,
    updated_at    timestamp,
    body          text
);

CREATE TABLE IF NOT EXISTS collections (
    id            text  PRIMARY KEY,
    name          text,
    slug          text,
    display_order int,
    product_ids   list<text>,
    created_at    timestamp,
    updated_at    timestamp,
    body          text
);
