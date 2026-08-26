-- name: ListCountries :many
SELECT c.id, c.code, c.name, n.locale, n.name AS locale_name
FROM openfaithmap.refdata_countries c
LEFT JOIN openfaithmap.refdata_country_names n ON n.code = c.code
ORDER BY c.sort_order NULLS LAST, c.code;
