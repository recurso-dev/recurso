from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.get_unit_economics_response_200 import GetUnitEconomicsResponse200
from ...types import Response


def _get_kwargs() -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/analytics/unit-economics",
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | GetUnitEconomicsResponse200 | None:
    if response.status_code == 200:
        response_200 = GetUnitEconomicsResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | GetUnitEconomicsResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[Error | GetUnitEconomicsResponse200]:
    """Unit economics (ARPA / ARPU / LTV)

     Average revenue per account (customer) and per subscription, plus LTV (ARPA ÷ monthly revenue
    churn). LTV is reported once MRR history exists and churn is non-zero (has_ltv).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetUnitEconomicsResponse200]
    """

    kwargs = _get_kwargs()

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
) -> Error | GetUnitEconomicsResponse200 | None:
    """Unit economics (ARPA / ARPU / LTV)

     Average revenue per account (customer) and per subscription, plus LTV (ARPA ÷ monthly revenue
    churn). LTV is reported once MRR history exists and churn is non-zero (has_ltv).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetUnitEconomicsResponse200
    """

    return sync_detailed(
        client=client,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
) -> Response[Error | GetUnitEconomicsResponse200]:
    """Unit economics (ARPA / ARPU / LTV)

     Average revenue per account (customer) and per subscription, plus LTV (ARPA ÷ monthly revenue
    churn). LTV is reported once MRR history exists and churn is non-zero (has_ltv).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetUnitEconomicsResponse200]
    """

    kwargs = _get_kwargs()

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
) -> Error | GetUnitEconomicsResponse200 | None:
    """Unit economics (ARPA / ARPU / LTV)

     Average revenue per account (customer) and per subscription, plus LTV (ARPA ÷ monthly revenue
    churn). LTV is reported once MRR history exists and churn is non-zero (has_ltv).

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetUnitEconomicsResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
        )
    ).parsed
