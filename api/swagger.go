//Package Kala Kala API
//
//Kala API
//
//
//    Schemes: http, https
//    Host: API_HOST
//    BasePath: /api/v1/
//    Version: 1.0.1
//
//    Consumes:
//     - multipart/form-data
//     - application/json
//
//    Produces:
//     - application/json
//
//swagger:meta
package api

//
// swagger:operation POST /job/ jobs createJob
//
// Creating a Job
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// parameters:
// - name: body
//   in: body
//   required: true
//   schema:
//     type: object
//     required:
//       - name
//       - type
//     properties:
//       name:
//         type: string
//       type:
//         type: number
//         enum: [0, 1]
//         default: 0
//       owner:
//         type: string
//       schedule:
//         type: string
//       retries:
//         type: number
//       epsilon:
//         type: string
//       remote_properties:
//         type: object
//         properties:
//            url:
//              type: string
//            method:
//              type: string
//            body:
//              type: string
//            headers:
//              type: array
//              items:
//                 type: object
//                 properties:
//                   key:
//                     type: string
//                   value:
//                     type: string
//            timeout:
//              type: number
//            expected_response_codes:
//              type: array
//              items:
//                 type: number
//              default: [200]
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//

//
// swagger:operation GET /job jobs getAllJobs
//
// Getting a list of all Jobs
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//

//
// swagger:operation GET /job/{id}/ jobs getJob
//
// Getting a Job
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// parameters:
// - name: id
//   in: path
//   required: true
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//

//
// swagger:operation DELETE /job/{id}/ jobs removeJob
//
// Deleting a Job
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// parameters:
// - name: id
//   in: path
//   required: true
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//

//
// swagger:operation DELETE /job/all/ jobs removeAllJob
//
// Deleting all Jobs
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//


//
// swagger:operation GET /job/stats/{id}/ jobs statsJob
//
// Getting metrics about a certain Job
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// parameters:
// - name: id
//   in: path
//   required: true
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//

//
// swagger:operation POST /job/start/{id}/ jobs startJob
//
// Starting a Job manually
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// parameters:
// - name: id
//   in: path
//   required: true
// responses:
//   204:
//     description: OK
//   500:
//     description: Error
//
//

//
// swagger:operation POST /job/disable/{id}/ jobs disableJob
//
// Disabling a Job
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// parameters:
// - name: id
//   in: path
//   required: true
// responses:
//   204:
//     description: OK
//   500:
//     description: Error
//
//

//
// swagger:operation POST /job/enable/{id}/ jobs enableJob
//
// Enabling a Job
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// parameters:
// - name: id
//   in: path
//   required: true
// responses:
//   204:
//     description: OK
//   500:
//     description: Error
//
//

//
// swagger:operation GET /stats/ stats statsAPP
//
// Getting app-level metrics
//
// ---
// consumes:
//   - application/json
// produces:
//   - application/json
// responses:
//   200:
//     description: OK
//   500:
//     description: Error
//
//