-- Table: public.kala_jobs

-- DROP TABLE public.kala_jobs;

CREATE TABLE kala_jobs
(
  name character varying(80),
  id character varying(40) NOT NULL,
  command character varying(1000),
  owner character varying(80),
  disabled character varying(5),
  dependent_jobs character varying(400),
  parent_jobs character varying(400),
  schedule character varying(80),
  retries integer,
  epsilon character varying(20),
  success_count integer,
  last_success timestamp without time zone,
  error_count integer,
  last_error timestamp without time zone,
  last_attempted_run timestamp without time zone,
  next_run_at timestamp without time zone,
  CONSTRAINT kala_jobs_pkey PRIMARY KEY (id)
)
