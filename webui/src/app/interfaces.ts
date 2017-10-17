export interface KalaStats {
  active_jobs: number;
  disabled_jobs: number;
  jobs: number;
  error_count: number;
  success_count: number;
  next_run_at: string;
  last_attempted_run: string;
  created: string;
}

export interface KalaStatResponse {
  Stats: KalaStats;
}

export interface JobStat {
  job_id: string
  ran_at: string
  number_of_retries: number
  output: string
  success: boolean
  execution_duration: number
}

export interface JobStatResponse {
  job_stats: JobStat[]
}

export interface RemotePropertiesHeaders {
  [key: string]: string[]
}

export interface RemoteProperties {
  headers?: RemotePropertiesHeaders
  headersStr?: string
  expected_response_codes?: number[]
  expected_response_codesStr?: string
  timeout?: number
}
export interface Job {
  id?: string
  name?: string
  stats?: JobStat[]
  type?: number
  remote_properties?: RemoteProperties
  disabled?: boolean
}

export interface ListJobs {
  [key:string]: Job
}

export interface ListJobsResponse {
  jobs: ListJobs;
}
export interface JobDetailResponse {
  job: Job;
}

export const LocalType = 0;
export const RemoteType = 1;

export interface headers {
  [key: string]: string[]
}

