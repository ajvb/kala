import {Component} from '@angular/core';
import {HttpClient} from "@angular/common/http";
import {MatSnackBar} from "@angular/material";
import {Job, LocalType, RemoteType, headers} from "./interfaces";

@Component({
  selector: 'create-job',
  templateUrl: './app.create.component.html',
  styleUrls: ['./app.create.component.css']
})

export class AppCreateComponent {
  methods = ["GET", "POST", "HEAD", "PUT", "DELETE", "CONNECT", "OPTIONS", "PATCH", "TRACE"];
  isRequesting = false;
  local_type = LocalType;
  remote_type = RemoteType;
  job: Job = {
    type: LocalType,
    remote_properties: {
      headers: {},
      expected_response_codesStr: "200,201",
      timeout: 10,
    },
  };

  constructor(
    private http: HttpClient,
    public snackBar: MatSnackBar,
  ){}

  create(): void {
    this.isRequesting = true;
    console.log(this.job);
    if (this.job.type == RemoteType) {
      if (this.job.remote_properties.headersStr != "") {
        const h = JSON.parse(this.job.remote_properties.headersStr);
        this.job.remote_properties.headers = h;
      }
      const codesStr = this.job.remote_properties.expected_response_codesStr.split(",");
      const codes = codesStr.map((value: string) => Number(value));
      this.job.remote_properties.expected_response_codes = codes;
    }

    this.http.post("http://localhost:8000/api/v1/job/", JSON.stringify(this.job)).subscribe(data => {
      console.log(data);

      this.snackBar.open("Created", "", {duration: 2500});
      this.isRequesting = false;
      this.job = {
        type: this.job.type,
        remote_properties: {
          headers: {},
          expected_response_codesStr: "200,201",
          timeout: 10,
        },
      };
    }, err => {
      console.log("Error occured", err.error);
      this.isRequesting = false;
      this.snackBar.open("Failed to create", "dismiss", {duration: 1000 * 5});
    })
  }
}
