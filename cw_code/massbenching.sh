for i in {1..5}
do
cat go test -bench /16x16x2 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /16x16x4 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /16x16x8 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /64x64x2 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /64x64x4 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /64x64x8 >> result.txt
sleep 20
done 


for i in {1..5}
do
cat go test -bench /128x128x2 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /128x128x4 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /128x128x8 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /256x256x2 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /256x256x4 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /256x256x8 >> result.txt
sleep 20
done


for i in {1..5}
do
cat go test -bench /512x512x2 >> result.txt
sleep 20
done



for i in {1..5}
do
cat go test -bench /512x512x4 >> result.txt
sleep 20
done



for i in {1..5}
do
cat go test -bench /512x512x8 >> result.txt
sleep 20
done


