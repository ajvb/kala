mysql -ukala -pkala < delete.sql
rm date.txt
cd ../..
go run main.go serve --jobdb=mariadb --jobdb-address=\(localhost:3306\)/kala --jobdb-username=kala --jobdb-password=kala &
cd deployment/automate-testing
#GO_PID=$!
ps -ef | grep kala | awk -F ' ' '{print $2}' > pid.txt
sleep 5
now=$(date --date="+15 seconds" +%Y-%m-%dT%H:%M:%S+02:00)
id=$(curl http://127.0.0.1:8000/api/v1/job/ -d '{"epsilon": "PT5S", "command": "bash ./deployment/automate-testing/example-command.sh", "name": "test_job", "schedule": "R/'$now'/PT5S"}' | jq -r .id)
sleep 17
initial=$(cat date.txt | wc -l)
curl -X DELETE http://127.0.0.1:8000/api/v1/job/$id/
sleep 13
final=$(cat date.txt | wc -l)
if [ $initial == $final ]; then
  echo "Strings are equal"
else
  echo "Strings are not equal"
fi
#kill $GO_PID
kill -9 `cat pid.txt`
rm pid.txt
