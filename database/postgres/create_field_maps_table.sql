CREATE TABLE IF NOT EXISTS FieldMaps (
    FieldName TEXT,
    ObjectType TEXT,
    JsonField TEXT,
    DataSetName TEXT,
    DataSetLabel TEXT,
    PRIMARY KEY (FieldName, ObjectType)
);
