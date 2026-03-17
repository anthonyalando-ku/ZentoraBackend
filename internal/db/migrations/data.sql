BEGIN;
INSERT INTO product_brands (name, slug, logo_url) VALUES
('Apple', 'apple', NULL),
('Sony', 'sony', NULL),
('LG', 'lg', NULL),
('Panasonic', 'panasonic', NULL),

('Nike', 'nike', NULL),
('Adidas', 'adidas', NULL),
('Puma', 'puma', NULL),
('Under Armour', 'under-armour', NULL),

('Philips', 'philips', NULL),
('Tefal', 'tefal', NULL),
('Black+Decker', 'black-decker', NULL),

('L''Oreal', 'loreal', NULL),
('Nivea', 'nivea', NULL),
('Maybelline', 'maybelline', NULL),

('Ikea', 'ikea', NULL),
('Home Basics', 'home-basics', NULL),

('Chicco', 'chicco', NULL),
('Johnson''s Baby', 'johnsons-baby', NULL),

('Oral-B', 'oral-b', NULL),
('Dettol', 'dettol', NULL),

('Bosch', 'bosch', NULL),
('Michelin', 'michelin', NULL),

('Logitech', 'logitech', NULL),
('HP', 'hp', NULL),
('Dell', 'dell', NULL),

('Penguin', 'penguin', NULL),
('Oxford', 'oxford', NULL);

COMMIT;

BEGIN;

INSERT INTO attributes (id, name, slug, is_variant_dimension) VALUES
(2, 'Color', 'color', TRUE),
(3, 'Storage', 'storage', TRUE),
(4, 'RAM', 'ram', TRUE),
(5, 'Material', 'material', FALSE),
(6, 'Weight', 'weight', FALSE),
(7, 'Volume', 'volume', FALSE),
(8, 'Power', 'power', FALSE),
(9, 'Gender', 'gender', FALSE),
(10, 'Age Group', 'age-group', FALSE),
(11, 'Language', 'language', FALSE),
(12, 'Cover Type', 'cover-type', FALSE),
(13, 'Screen Size', 'screen-size', TRUE),
(14, 'Connectivity', 'connectivity', FALSE),
(15, 'Compatibility', 'compatibility', FALSE);

COMMIT;

INSERT INTO attribute_values (attribute_id, value) VALUES
(2,'Black'),
(2,'White'),
(2,'Red'),
(2,'Blue'),
(2,'Green'),
(2,'Gray'),
(2,'Silver'),
(2,'Gold');

INSERT INTO attribute_values (attribute_id, value) VALUES
(1,'XS'),
(1,'S'),
(1,'M'),
(1,'L'),
(1,'XL'),
(1,'XXL');

INSERT INTO attribute_values (attribute_id, value) VALUES
(3,'32GB'),
(3,'64GB'),
(3,'128GB'),
(3,'256GB'),
(3,'512GB'),
(3,'1TB');


INSERT INTO attribute_values (attribute_id, value) VALUES
(4,'2GB'),
(4,'4GB'),
(4,'8GB'),
(4,'16GB'),
(4,'32GB');

INSERT INTO attribute_values (attribute_id, value) VALUES
(5,'Cotton'),
(5,'Leather'),
(5,'Plastic'),
(5,'Metal'),
(5,'Glass'),
(5,'Wood'),
(5,'Stainless Steel');

INSERT INTO attribute_values (attribute_id, value) VALUES
(6,'100g'),
(6,'250g'),
(6,'500g'),
(6,'1kg'),
(6,'2kg');

INSERT INTO attribute_values (attribute_id, value) VALUES
(7,'100ml'),
(7,'250ml'),
(7,'500ml'),
(7,'750ml'),
(7,'1L');

INSERT INTO attribute_values (attribute_id, value) VALUES
(8,'500W'),
(8,'750W'),
(8,'1000W'),
(8,'1500W'),
(8,'2000W');

INSERT INTO attribute_values (attribute_id, value) VALUES
(9,'Men'),
(9,'Women'),
(9,'Unisex');

INSERT INTO attribute_values (attribute_id, value) VALUES
(10,'Newborn'),
(10,'0-6 Months'),
(10,'6-12 Months'),
(10,'1-3 Years'),
(10,'3+ Years');

INSERT INTO attribute_values (attribute_id, value) VALUES
(11,'English'),
(11,'French'),
(11,'Spanish');

INSERT INTO attribute_values (attribute_id, value) VALUES
(12,'Paperback'),
(12,'Hardcover');

INSERT INTO attribute_values (attribute_id, value) VALUES
(13,'13 inch'),
(13,'15 inch'),
(13,'17 inch'),
(13,'24 inch'),
(13,'27 inch');

INSERT INTO attribute_values (attribute_id, value) VALUES
(14,'Bluetooth'),
(14,'WiFi'),
(14,'USB-C'),
(14,'HDMI');

