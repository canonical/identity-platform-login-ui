package authorization

// Code generated by Makefile; DO NOT EDIT.

var AuthModel = `{"schema_version":"1.1","type_definitions":[{"type":"user"},{"type":"app"},{"metadata":{"relations":{"member":{"directly_related_user_types":[{"type":"user"},{"relation":"member","type":"group"}]}}},"relations":{"member":{"this":{}}},"type":"group"},{"metadata":{"relations":{"member":{"directly_related_user_types":[{"type":"app"},{"relation":"member","type":"app_group"}]}}},"relations":{"member":{"this":{}}},"type":"app_group"},{"metadata":{"relations":{"allowed_access":{"directly_related_user_types":[{"type":"app"},{"relation":"member","type":"app_group"}]},"member":{"directly_related_user_types":[{"type":"user"}]}}},"relations":{"allowed_access":{"this":{}},"member":{"this":{}}},"type":"provider"}]}`
