import {Injectable} from "@angular/core";
import {HttpClient} from "@angular/common/http";
import {Job, JobDetailResponse, JobStatResponse, KalaStatResponse, KalaStats, ListJobsResponse} from "./interfaces";
import {Observable} from "rxjs/Observable";
import {environment} from "../environments/environment";

const KalaHost = environment.kalaHost;
const Version = 'v1';

const ApiUriPrefix = KalaHost + "/api/" + Version + "/";
const ApiJobUri = ApiUriPrefix + "job/";

@Injectable()
export class KalaService {
  constructor(
    public http: HttpClient
  ) { }

  getMetrics(): Observable<KalaStatResponse> {
    return this.http.get<KalaStatResponse>(ApiUriPrefix + "stats/");
  }

  getJobDetail(id: string): Observable<JobDetailResponse> {
      return this.http.get<JobDetailResponse>(ApiJobUri + id +"/");
  }

  getJobStats(id: string): Observable<JobStatResponse> {
    return this.http.get<JobStatResponse>(ApiJobUri + "stats/" +id +"/");
  }

  getJobs(): Observable<ListJobsResponse> {
    return this.http.get<ListJobsResponse>(ApiJobUri);
  }

  deleteJob(id: string): Observable<Object> {
    return this.http.delete(ApiJobUri + id + "/");
  }

  startJob(id: string): Observable<Object> {
    return this.http.post(ApiJobUri + "start/" + id + "/", null);
  }

  enableJob(id: string): Observable<Object> {
    return this.http.post(ApiJobUri + "enable/" +id + "/", null);
  }

  disableJob(id: string): Observable<Object> {
    return this.http.post(ApiJobUri + "disable/" +id + "/", null);
  }

  createJob(job: Job): Observable<Object> {
    return this.http.post(ApiJobUri, JSON.stringify(job));
  }
}
