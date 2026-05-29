for file in api/db/**/*.sql; do
  npx sql-formatter --fix -l sqlite "$file"
done
