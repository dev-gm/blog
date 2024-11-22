rm blog.db
cd web
npm run build
cd ..
cp web/dist/index.html views/nested/index.html
go build
./blog
