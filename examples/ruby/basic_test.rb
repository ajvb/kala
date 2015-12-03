require 'net/http'
require 'time'
require 'json'

class Kala
  API_URL = 'http://127.0.0.1:8000/api/v1/job/'

  def post(data)
    req = Net::HTTP::Post.new(uri.path)
    req.body = data.to_json
    call(req)
  end

  def get(id)
    req = Net::HTTP::Get.new("#{uri.path}#{id}/")
    call(req)
  end

  def delete(id)
      req = Net::HTTP::Delete.new("#{uri.path}#{id}/")
    call(req)
  end

  private

  def uri
    @uri ||= URI(API_URL)
  end

  def call(req)
    res = Net::HTTP.start(uri.host, uri.port) do |http|
      http.request(req)
    end

    if res.body
      JSON.parse(res.body)
    else
      res
    end
  end
end

data = {
  name: 'ruby_job',
  command: "bash #{Dir.pwd}/../example-kala-commands/example-command.sh",
  epsilon: 'PT5S',
  schedule: "R2/#{(Time.now + 10).iso8601}/PT10S"
}

puts "Sending request to #{Kala::API_URL}"
puts "Payload is: #{data}\n\n"

kala = Kala.new

job_id = kala.post(data)['id']
puts "Job was created with an id of #{job_id}"

puts "Getting informations about job #{job_id}"
puts kala.get(job_id)

puts "Waiting to delete job #{job_id}"
sleep(21)
puts kala.delete(job_id).inspect
