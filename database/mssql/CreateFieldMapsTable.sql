IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='FieldMaps' and xtype='U')
CREATE TABLE FieldMaps (
    FieldName NVARCHAR(255),
    ObjectType NVARCHAR(255),
    JsonField NVARCHAR(255),
    DataSetName NVARCHAR(255),
    DataSetLabel NVARCHAR(255),
    PRIMARY KEY (FieldName, ObjectType)
);
