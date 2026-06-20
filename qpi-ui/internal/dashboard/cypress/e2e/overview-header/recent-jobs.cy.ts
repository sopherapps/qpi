describe("Overview & Header — Recent Jobs Table", () => {
  beforeEach(() => {
    cy.clearCookies();
    cy.clearLocalStorage();
    cy.visit("/");

    cy.get('input[type="text"]').clear().type("user@example.com");
    cy.get('input[type="password"]').clear().type("userpassword1234");
    cy.get('button[type="submit"]').click();
    cy.contains("h1", "QPI Interface").should("be.visible");
  });

  it("shows actual job data from the API", () => {
    cy.window().then((win) => {
      const authStoreStr = win.localStorage.getItem("pocketbase_auth");
      const authStore = authStoreStr ? JSON.parse(authStoreStr) : {};
      const token = authStore.token || "";

      cy.request({
        method: "GET",
        url: "http://127.0.0.1:8090/api/collections/quantum_jobs/records?sort=-created",
        headers: { Authorization: token },
      }).then((res) => {
        const jobs = res.body.items || [];

        if (jobs.length === 0) {
          cy.contains("No jobs submitted yet.").should("be.visible");
          return;
        }

        // The first job in the table should match the first job from the API
        const firstJob = jobs[0];
        cy.get("table tbody tr")
          .first()
          .within(() => {
            cy.get("td").eq(0).should("contain", firstJob.id);
            cy.get("td").eq(2).should("contain", firstJob.status);
          });
      });
    });
  });

  it("shows the correct QPU name for each job", () => {
    cy.window().then((win) => {
      const authStoreStr = win.localStorage.getItem("pocketbase_auth");
      const authStore = authStoreStr ? JSON.parse(authStoreStr) : {};
      const token = authStore.token || "";

      cy.request({
        method: "GET",
        url: "http://127.0.0.1:8090/api/collections/qpus/records",
        headers: { Authorization: token },
      }).then((qpuRes) => {
        const qpus = qpuRes.body.items || [];

        cy.request({
          method: "GET",
          url: "http://127.0.0.1:8090/api/collections/quantum_jobs/records?sort=-created",
          headers: { Authorization: token },
        }).then((jobRes) => {
          const jobs = jobRes.body.items || [];

          if (jobs.length === 0) return;

          const firstJob = jobs[0];
          const targetQpu = qpus.find((q: { id: string }) => q.id === firstJob.qpu_target);
          const expectedName = targetQpu?.name || firstJob.qpu_target;

          cy.get("table tbody tr")
            .first()
            .within(() => {
              cy.get("td").eq(1).should("contain", expectedName);
            });
        });
      });
    });
  });
});
