from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.get_tax_nexus_status_response_200 import GetTaxNexusStatusResponse200
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    year: int | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["year"] = year

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/settings/tax/nexus/status",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | GetTaxNexusStatusResponse200 | None:
    if response.status_code == 200:
        response_200 = GetTaxNexusStatusResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | GetTaxNexusStatusResponse200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    year: int | Unset = UNSET,
) -> Response[Error | GetTaxNexusStatusResponse200]:
    """Per-state US economic-nexus status

     Returns, for the given calendar year (default current), every US state where the tenant has declared
    or economic nexus or any US sales activity: year-to-date taxable sales and transaction counts, the
    state's economic-nexus threshold, proximity to it (percent), and whether it is crossed. Crossings
    detected during the read are auto-established as economic nexus (idempotent). dataset_certified is
    false until the seeded threshold dataset passes professional review — callers must surface that
    caveat before the data is relied on for filing.

    Args:
        year (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetTaxNexusStatusResponse200]
    """

    kwargs = _get_kwargs(
        year=year,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    year: int | Unset = UNSET,
) -> Error | GetTaxNexusStatusResponse200 | None:
    """Per-state US economic-nexus status

     Returns, for the given calendar year (default current), every US state where the tenant has declared
    or economic nexus or any US sales activity: year-to-date taxable sales and transaction counts, the
    state's economic-nexus threshold, proximity to it (percent), and whether it is crossed. Crossings
    detected during the read are auto-established as economic nexus (idempotent). dataset_certified is
    false until the seeded threshold dataset passes professional review — callers must surface that
    caveat before the data is relied on for filing.

    Args:
        year (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetTaxNexusStatusResponse200
    """

    return sync_detailed(
        client=client,
        year=year,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    year: int | Unset = UNSET,
) -> Response[Error | GetTaxNexusStatusResponse200]:
    """Per-state US economic-nexus status

     Returns, for the given calendar year (default current), every US state where the tenant has declared
    or economic nexus or any US sales activity: year-to-date taxable sales and transaction counts, the
    state's economic-nexus threshold, proximity to it (percent), and whether it is crossed. Crossings
    detected during the read are auto-established as economic nexus (idempotent). dataset_certified is
    false until the seeded threshold dataset passes professional review — callers must surface that
    caveat before the data is relied on for filing.

    Args:
        year (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetTaxNexusStatusResponse200]
    """

    kwargs = _get_kwargs(
        year=year,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    year: int | Unset = UNSET,
) -> Error | GetTaxNexusStatusResponse200 | None:
    """Per-state US economic-nexus status

     Returns, for the given calendar year (default current), every US state where the tenant has declared
    or economic nexus or any US sales activity: year-to-date taxable sales and transaction counts, the
    state's economic-nexus threshold, proximity to it (percent), and whether it is crossed. Crossings
    detected during the read are auto-established as economic nexus (idempotent). dataset_certified is
    false until the seeded threshold dataset passes professional review — callers must surface that
    caveat before the data is relied on for filing.

    Args:
        year (int | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetTaxNexusStatusResponse200
    """

    return (
        await asyncio_detailed(
            client=client,
            year=year,
        )
    ).parsed
