describe("Overview & Header — Metrics Row", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it("displays metrics that match the actual API data", () => {
    cy.window().then((win) => {
      const authStoreStr = win.localStorage.getItem("pocketbase_auth");
      const authStore = authStoreStr ? JSON.parse(authStoreStr) : {};
      const token = authStore.token || "";

      cy.request({
        method: "GET",
        url: "http://127.0.0.1:8090/api/collections/qpus/records",
        headers: { Authorization: token },
      }).then((qpuRes) => {
        const qpus = Array.isArray(qpuRes.body) ? qpuRes.body : (qpuRes.body.items || []);
        const onlineQpus = qpus.filter((q: { status: string }) => q.status === "online").length;

        cy.request({
          method: "GET",
          url: "http://127.0.0.1:8090/api/collections/quantum_jobs/records",
          headers: { Authorization: token },
        }).then((jobRes) => {
          const jobs = jobRes.body.items || [];
          const pendingJobs = jobs.filter((j: { status: string }) => j.status === "pending").length;
          const runningJobs = jobs.filter((j: { status: string }) => j.status === "running").length;

          cy.request({
            method: "GET",
            url: "http://127.0.0.1:8090/api/collections/time_slots/records",
            headers: { Authorization: token },
          }).then((bookingRes) => {
            const bookings = bookingRes.body.items || [];

            cy.request({
              method: "GET",
              url: "http://127.0.0.1:8090/api/collections/users/records",
              headers: { Authorization: token },
            }).then((userRes) => {
              const users = userRes.body.items || [];
              const currentUser = users.find((u: { email: string }) => u.email === "user@example.com");
              const qpuSeconds = currentUser?.qpu_seconds || 0;

              // Active QPUs card
              cy.contains("Active QPUs")
                .parent()
                .parent()
                .within(() => {
                  cy.get("div.text-2xl").should("contain", `${onlineQpus}/${qpus.length}`);
                });

              // Queue Status card
              cy.contains("Queue Status")
                .parent()
                .parent()
                .within(() => {
                  cy.get("div.text-2xl").should("contain", `${pendingJobs + runningJobs} jobs`);
                  cy.contains(`${pendingJobs} pending, ${runningJobs} running`).should("be.visible");
                });

              // Time Credit card
              cy.contains("Time Credit")
                .parent()
                .parent()
                .within(() => {
                  cy.get("div.text-2xl").should("contain", `${qpuSeconds}s`);
                });

              // Next Booking card
              cy.contains("Next Booking")
                .parent()
                .parent()
                .within(() => {
                  const futureBookings = bookings
                    .filter((b: { start_time: string }) => new Date(b.start_time) > new Date())
                    .sort((a: { start_time: string }, b: { start_time: string }) =>
                      new Date(a.start_time).getTime() - new Date(b.start_time).getTime()
                    );
                  if (futureBookings.length > 0) {
                    cy.get("div.text-lg").should("not.contain", "None Scheduled");
                  } else {
                    cy.get("div.text-lg").should("contain", "None Scheduled");
                  }
                });
            });
          });
        });
      });
    });
  });

  it("navigates to the corresponding tab when clicking on a card", () => {
    cy.contains("Active QPUs").parent().parent().click();
    cy.hash().should("eq", "#qpus");

    cy.get('nav button:contains("Overview")').click();

    cy.contains("Queue Status").parent().parent().click();
    cy.hash().should("eq", "#jobs");

    cy.get('nav button:contains("Overview")').click();

    cy.contains("Next Booking").parent().parent().click();
    cy.hash().should("eq", "#bookings");
  });
});
